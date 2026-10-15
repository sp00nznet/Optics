// Optics.exe: pick a folder, see every model in it. One window: the path and
// a filter across the top, the file list on the left, the 3D view (or the
// source of a .scad) on the right. Nothing is stored except the last folder.

#include "optics.h"

#include <commctrl.h>
#include <shellapi.h>
#include <shlobj.h>
#include <shlwapi.h>
#include <uxtheme.h>
#include <windowsx.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

enum { IDC_PATH = 100, IDC_BROWSE, IDC_RESCAN, IDC_FILTER, IDC_LIST, IDC_SOURCE, IDC_STATUS };
#define WM_APP_SCANNED (WM_APP + 1)
#define WM_APP_THUMB (WM_APP + 2)
#define TIMER_PROGRESS 1

static HWND g_main, g_path, g_browse, g_rescan, g_filter, g_list, g_view, g_source, g_status;
static HFONT g_font, g_mono;
static HGLRC g_rc;

static Library* g_lib;       // what the list shows
static Library* g_scanning;  // the scan in progress, if any
static int* g_shown;         // indices into g_lib->items after the filter
static int g_nshown, g_sortcol, g_sortdesc;
static Model g_model;
static BOOL g_hasModel;
static Camera g_cam = { 35, 25, 1.05f };
static char* g_select;       // a file named on the command line, picked once scanned

static int Px(int x) { return MulDiv(x, GetDpiForWindow(g_main), 96); }

static void SetStatus(int part, const char* text) {
    wchar_t* w = Widen(text);
    SendMessageW(g_status, SB_SETTEXTW, part, (LPARAM)(w ? w : L""));
    free(w);
}

// ---------------------------------------------------------------------------
// The 3D view
// ---------------------------------------------------------------------------

static void DropModel(void) {
    if (!g_hasModel) return;
    HDC dc = GetDC(g_view);
    wglMakeCurrent(dc, g_rc);
    Render_Forget(&g_model);
    ReleaseDC(g_view, dc);
    Model_Free(&g_model);
    g_hasModel = FALSE;
}

static LRESULT CALLBACK ViewProc(HWND h, UINT msg, WPARAM wp, LPARAM lp) {
    static POINT last;
    switch (msg) {
    case WM_CREATE: {
        HDC dc = GetDC(h);
        PIXELFORMATDESCRIPTOR pfd = { sizeof(pfd), 1, PFD_DRAW_TO_WINDOW | PFD_SUPPORT_OPENGL | PFD_DOUBLEBUFFER,
                                      PFD_TYPE_RGBA, 32 };
        pfd.cDepthBits = 24;
        SetPixelFormat(dc, ChoosePixelFormat(dc, &pfd), &pfd);
        g_rc = wglCreateContext(dc);
        ReleaseDC(h, dc);
        return g_rc ? 0 : -1;
    }
    case WM_ERASEBKGND:
        return 1;
    case WM_PAINT: {
        PAINTSTRUCT ps;
        HDC dc = BeginPaint(h, &ps);
        RECT rc;
        GetClientRect(h, &rc);
        wglMakeCurrent(dc, g_rc);
        Render_Draw(g_hasModel ? &g_model : NULL, &g_cam, rc.right, rc.bottom);
        SwapBuffers(dc);
        EndPaint(h, &ps);
        return 0;
    }
    case WM_LBUTTONDOWN:
        SetCapture(h);
        last.x = GET_X_LPARAM(lp);
        last.y = GET_Y_LPARAM(lp);
        return 0;
    case WM_MOUSEMOVE:
        if (wp & MK_LBUTTON) {
            g_cam.yaw += (GET_X_LPARAM(lp) - last.x) * 0.5f;
            g_cam.pitch += (GET_Y_LPARAM(lp) - last.y) * 0.5f;
            if (g_cam.pitch > 89) g_cam.pitch = 89;
            if (g_cam.pitch < -89) g_cam.pitch = -89;
            last.x = GET_X_LPARAM(lp);
            last.y = GET_Y_LPARAM(lp);
            InvalidateRect(h, NULL, FALSE);
        }
        return 0;
    case WM_LBUTTONUP:
        ReleaseCapture();
        return 0;
    case WM_LBUTTONDBLCLK:
        g_cam = (Camera){ 35, 25, 1.05f };
        InvalidateRect(h, NULL, FALSE);
        return 0;
    case WM_MOUSEWHEEL:
        g_cam.zoom *= GET_WHEEL_DELTA_WPARAM(wp) > 0 ? 0.9f : 1.1f;
        if (g_cam.zoom < 0.05f) g_cam.zoom = 0.05f;
        if (g_cam.zoom > 20) g_cam.zoom = 20;
        InvalidateRect(h, NULL, FALSE);
        return 0;
    case WM_DESTROY:
        DropModel();
        wglMakeCurrent(NULL, NULL);
        wglDeleteContext(g_rc);
        return 0;
    }
    return DefWindowProcW(h, msg, wp, lp);
}

// ---------------------------------------------------------------------------
// Showing one item
// ---------------------------------------------------------------------------

static Bytes ReadItem(const Item* it) {
    Bytes b = { 0 };
    if (!it->inner) return ReadWholeFile(it->path);
    ZipIndex z;
    if (!Zip_List(it->path, &z)) return b;
    for (int i = 0; i < z.count; i++)
        if (strcmp(z.entries[i].name, it->inner) == 0) { b = Zip_Read(it->path, &z.entries[i]); break; }
    Zip_Free(&z);
    return b;
}

static void ShowSource(const Item* it) {
    Bytes b = ReadItem(it);
    // The edit control wants CRLF; .scad files are mostly LF.
    char* crlf = b.data ? malloc(b.len * 2 + 1) : NULL;
    size_t n = 0;
    for (size_t i = 0; crlf && i < b.len; i++) {
        if (b.data[i] == '\n' && (i == 0 || b.data[i - 1] != '\r')) crlf[n++] = '\r';
        crlf[n++] = b.data[i];
    }
    if (crlf) crlf[n] = 0;
    wchar_t* w = crlf ? Widen(crlf) : NULL;
    SetWindowTextW(g_source, w ? w : L"(could not read the file)");
    free(w);
    free(crlf);
    free(b.data);
}

static void ShowItem(const Item* it) {
    char status[600];
    DropModel();
    BOOL source = it->kind == K_SCAD;
    ShowWindow(g_source, source ? SW_SHOW : SW_HIDE);
    ShowWindow(g_view, source ? SW_HIDE : SW_SHOW);
    if (source) {
        ShowSource(it);
        sprintf_s(status, sizeof(status), "%s: OpenSCAD source", it->name);
    } else if (it->kind == K_3MF || it->kind == K_STEP) {
        sprintf_s(status, sizeof(status), "%s: no preview for %s files", it->name, kKindName[it->kind]);
    } else {
        HCURSOR old = SetCursor(LoadCursor(NULL, IDC_WAIT));
        g_hasModel = Model_Load(it, &g_model);
        SetCursor(old);
        if (!g_hasModel)
            sprintf_s(status, sizeof(status), "%s: %s", it->name, g_model.error);
        else if (g_model.tex_wanted)
            sprintf_s(status, sizeof(status), "%s: %lld triangles, textures %d of %d%s%s", it->name,
                      g_model.triangles, g_model.tex_loaded, g_model.tex_wanted,
                      g_model.missing[0] ? ", missing " : "", g_model.missing);
        else
            sprintf_s(status, sizeof(status), "%s: %lld triangles", it->name, g_model.triangles);
    }
    SetStatus(1, status);
    InvalidateRect(g_view, NULL, FALSE);
}

// ---------------------------------------------------------------------------
// Thumbnails: rendered only when the list asks for a row, newest request
// first, on one worker thread with the software GL. The rows on screen are
// always the newest requests, so they render first however far you scroll.
// ---------------------------------------------------------------------------

typedef struct {
    int gen, index;  // which library (g_gen) and which item in it
    Item item;       // copied: the library may be freed by a rescan meanwhile
    HBITMAP bmp;
} ThumbJob;

static HIMAGELIST g_thumbs;  // image 0 is blank, for rows without one yet
static int g_thumbsize;
static int* g_thumb;         // per g_lib item: 0 not asked, -1 none/pending, else image index
static int g_gen;
static ThumbJob** g_queue;
static int g_nqueue, g_capqueue;
static CRITICAL_SECTION g_qlock;
static HANDLE g_qwake;

static void FreeJob(ThumbJob* j) {
    free(j->item.path); free(j->item.inner); free(j->item.name); free(j->item.rel);
    if (j->bmp) DeleteObject(j->bmp);
    free(j);
}

static HBITMAP Thumbnail(const Item* it) {
    Model m;
    if (!Model_Load(it, &m)) return NULL;
    int s = g_thumbsize, big = s * 2;  // 2x2 supersampled: the software GL has no antialiasing
    Camera cam = { 35, 25, 1.05f };
    unsigned char* px = Render_Offscreen(&m, &cam, big, big);
    Model_Free(&m);
    if (!px) return NULL;
    BITMAPINFO bi = { { sizeof(BITMAPINFOHEADER), s, -s, 1, 32, BI_RGB } };
    unsigned char* bits = NULL;
    HBITMAP bmp = CreateDIBSection(NULL, &bi, DIB_RGB_COLORS, (void**)&bits, NULL, 0);
    for (int y = 0; bmp && y < s; y++)
        for (int x = 0; x < s; x++) {
            const unsigned char* a = px + ((size_t)(2 * y) * big + 2 * x) * 4;
            const unsigned char* b = a + (size_t)big * 4;
            unsigned char* d = bits + ((size_t)y * s + x) * 4;
            for (int c = 0; c < 3; c++) d[2 - c] = (unsigned char)((a[c] + a[c + 4] + b[c] + b[c + 4] + 2) / 4);
            d[3] = 255;
        }
    free(px);
    return bmp;
}

static DWORD WINAPI ThumbThread(void* p) {
    (void)p;
    for (;;) {
        WaitForSingleObject(g_qwake, INFINITE);
        for (;;) {
            EnterCriticalSection(&g_qlock);
            ThumbJob* j = g_nqueue ? g_queue[--g_nqueue] : NULL;
            LeaveCriticalSection(&g_qlock);
            if (!j) break;
            j->bmp = Thumbnail(&j->item);
            if (!PostMessageW(g_main, WM_APP_THUMB, 0, (LPARAM)j)) FreeJob(j);
        }
    }
}

// Also on a DPI change: a window often starts on one monitor and lands on another.
static void MakeThumbList(void) {
    if (g_thumbs) ImageList_Destroy(g_thumbs);
    g_thumbsize = Px(48);
    g_thumbs = ImageList_Create(g_thumbsize, g_thumbsize, ILC_COLOR32, 64, 64);
    BITMAPINFO bi = { { sizeof(BITMAPINFOHEADER), g_thumbsize, -g_thumbsize, 1, 32, BI_RGB } };
    DWORD* bits = NULL;
    HBITMAP blank = CreateDIBSection(NULL, &bi, DIB_RGB_COLORS, (void**)&bits, NULL, 0);
    // The list draws this opaque, so it is the list's own background colour.
    COLORREF bg = GetSysColor(COLOR_WINDOW);
    for (int i = 0; bits && i < g_thumbsize * g_thumbsize; i++)
        bits[i] = 0xFF000000u | (GetRValue(bg) << 16) | (GetGValue(bg) << 8) | GetBValue(bg);
    ImageList_Add(g_thumbs, blank, NULL);
    DeleteObject(blank);
    ListView_SetImageList(g_list, g_thumbs, LVSIL_SMALL);
}

static void StartThumbs(void) {
    MakeThumbList();
    InitializeCriticalSection(&g_qlock);
    g_qwake = CreateEventW(NULL, FALSE, FALSE, NULL);
    HANDLE t = CreateThread(NULL, 0, ThumbThread, NULL, 0, NULL);
    if (t) CloseHandle(t);
}

// A new library: drop the old thumbnails and anything still queued for them.
static void ResetThumbs(void) {
    g_gen++;
    EnterCriticalSection(&g_qlock);
    for (int i = 0; i < g_nqueue; i++) FreeJob(g_queue[i]);
    g_nqueue = 0;
    LeaveCriticalSection(&g_qlock);
    ImageList_SetImageCount(g_thumbs, 1);
    free(g_thumb);
    g_thumb = g_lib ? calloc(g_lib->count ? g_lib->count : 1, sizeof(int)) : NULL;
    InvalidateRect(g_list, NULL, FALSE);
}

static int ThumbFor(int index) {
    if (!g_thumb) return 0;
    if (g_thumb[index] == 0) {
        g_thumb[index] = -1;
        Kind k = g_lib->items[index].kind;
        ThumbJob* j = k == K_STL || k == K_OBJ || k == K_FBX ? calloc(1, sizeof(*j)) : NULL;
        if (j) {
            const Item* it = &g_lib->items[index];
            j->gen = g_gen;
            j->index = index;
            j->item = *it;
            j->item.path = _strdup(it->path);
            j->item.inner = it->inner ? _strdup(it->inner) : NULL;
            j->item.name = _strdup(it->name);
            j->item.rel = _strdup(it->rel);
            EnterCriticalSection(&g_qlock);
            if (g_nqueue == g_capqueue) {
                int cap = g_capqueue ? g_capqueue * 2 : 64;
                ThumbJob** grown = realloc(g_queue, cap * sizeof(*grown));
                if (grown) { g_queue = grown; g_capqueue = cap; }
            }
            if (g_nqueue < g_capqueue) { g_queue[g_nqueue++] = j; j = NULL; }
            LeaveCriticalSection(&g_qlock);
            if (j) FreeJob(j);
            else SetEvent(g_qwake);
        }
    }
    return g_thumb[index] > 0 ? g_thumb[index] : 0;
}

static void ThumbDone(ThumbJob* j) {
    if (j->gen == g_gen && g_thumb) {
        int image = j->bmp ? ImageList_Add(g_thumbs, j->bmp, NULL) : -1;
        g_thumb[j->index] = image > 0 ? image : -1;
        if (image > 0) InvalidateRect(g_list, NULL, FALSE);
    }
    FreeJob(j);
}

// ---------------------------------------------------------------------------
// The list
// ---------------------------------------------------------------------------

static int CompareItems(const void* a, const void* b) {
    const Item* x = &g_lib->items[*(const int*)a];
    const Item* y = &g_lib->items[*(const int*)b];
    int r = 0;
    switch (g_sortcol) {
    case 0: r = _stricmp(x->name, y->name); break;
    case 1: r = (int)x->kind - (int)y->kind; break;
    case 3: r = x->size < y->size ? -1 : x->size > y->size; break;
    }
    if (!r) r = _stricmp(x->rel, y->rel);
    return g_sortdesc ? -r : r;
}

static void Refilter(void) {
    wchar_t wf[256];
    GetWindowTextW(g_filter, wf, 256);
    char* filter = Narrow(wf);
    free(g_shown);
    g_shown = NULL;
    g_nshown = 0;
    if (g_lib && g_lib->count) {
        g_shown = malloc(g_lib->count * sizeof(int));
        for (int i = 0; g_shown && i < g_lib->count; i++)
            if (!filter || !filter[0] || StrStrIA(g_lib->items[i].rel, filter))
                g_shown[g_nshown++] = i;
        if (g_shown) qsort(g_shown, g_nshown, sizeof(int), CompareItems);
    }
    free(filter);
    ListView_SetItemCountEx(g_list, g_nshown, 0);
    ListView_SetItemState(g_list, -1, 0, LVIS_SELECTED);
    InvalidateRect(g_list, NULL, TRUE);
    char text[64];
    sprintf_s(text, sizeof(text), "%d models, %d shown", g_lib ? g_lib->count : 0, g_nshown);
    SetStatus(0, text);
}

static void GetCell(int row, int col, wchar_t* out, int cap) {
    out[0] = 0;
    if (row < 0 || row >= g_nshown) return;
    const Item* it = &g_lib->items[g_shown[row]];
    char buf[64];
    const char* s = buf;
    switch (col) {
    case 0: s = it->name; break;
    case 1: s = kKindName[it->kind]; break;
    case 2: s = it->rel; break;
    case 3: {
        double kb = it->size / 1024.0;
        if (kb < 1024) sprintf_s(buf, sizeof(buf), "%.0f KB", kb < 1 ? 1 : kb);
        else sprintf_s(buf, sizeof(buf), "%.1f MB", kb / 1024);
        break;
    }
    }
    MultiByteToWideChar(CP_UTF8, 0, s, -1, out, cap);
}

// ---------------------------------------------------------------------------
// Scanning, on a thread so a big library does not freeze the window
// ---------------------------------------------------------------------------

typedef struct {
    Library* lib;
    char* root;
} ScanJob;

static DWORD WINAPI ScanThread(void* p) {
    ScanJob* job = p;
    Scan(job->root, job->lib);
    PostMessageW(g_main, WM_APP_SCANNED, 0, (LPARAM)job->lib);
    free(job->root);
    free(job);
    return 0;
}

static void StartScan(void) {
    wchar_t wroot[MAX_PATH * 2];
    GetWindowTextW(g_path, wroot, MAX_PATH * 2);
    if (!wroot[0]) return;
    RegSetKeyValueW(HKEY_CURRENT_USER, L"Software\\Optics", L"LastPath", REG_SZ, wroot,
                    (DWORD)(wcslen(wroot) + 1) * sizeof(wchar_t));
    if (g_scanning) g_scanning->cancel = 1;  // its result is dropped when it lands
    ScanJob* job = malloc(sizeof(*job));
    if (!job) return;
    job->lib = calloc(1, sizeof(Library));
    job->root = Narrow(wroot);
    g_scanning = job->lib;
    HANDLE t = job->lib && job->root ? CreateThread(NULL, 0, ScanThread, job, 0, NULL) : NULL;
    if (t) CloseHandle(t);
    SetTimer(g_main, TIMER_PROGRESS, 150, NULL);
}

static void SetRoot(const wchar_t* path) {
    SetWindowTextW(g_path, path);
    StartScan();
}

static void Browse(void) {
    IFileOpenDialog* dlg;
    if (FAILED(CoCreateInstance(&CLSID_FileOpenDialog, NULL, CLSCTX_INPROC_SERVER, &IID_IFileOpenDialog,
                                (void**)&dlg)))
        return;
    DWORD opts;
    dlg->lpVtbl->GetOptions(dlg, &opts);
    dlg->lpVtbl->SetOptions(dlg, opts | FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM);
    dlg->lpVtbl->SetTitle(dlg, L"Choose a folder of models");
    IShellItem* item;
    if (SUCCEEDED(dlg->lpVtbl->Show(dlg, g_main)) && SUCCEEDED(dlg->lpVtbl->GetResult(dlg, &item))) {
        wchar_t* path;
        if (SUCCEEDED(item->lpVtbl->GetDisplayName(item, SIGDN_FILESYSPATH, &path))) {
            SetRoot(path);
            CoTaskMemFree(path);
        }
        item->lpVtbl->Release(item);
    }
    dlg->lpVtbl->Release(dlg);
}

// ---------------------------------------------------------------------------
// The main window
// ---------------------------------------------------------------------------

static void MakeFonts(void) {
    if (g_font) DeleteObject(g_font);
    if (g_mono) DeleteObject(g_mono);
    int dpi = GetDpiForWindow(g_main);
    g_font = CreateFontW(-MulDiv(9, dpi, 72), 0, 0, 0, FW_NORMAL, 0, 0, 0, DEFAULT_CHARSET, 0, 0, CLEARTYPE_QUALITY,
                         0, L"Segoe UI");
    g_mono = CreateFontW(-MulDiv(10, dpi, 72), 0, 0, 0, FW_NORMAL, 0, 0, 0, DEFAULT_CHARSET, 0, 0,
                         CLEARTYPE_QUALITY, FIXED_PITCH, L"Consolas");
    HWND controls[] = { g_path, g_browse, g_rescan, g_filter, g_list, g_status };
    for (int i = 0; i < 6; i++) SendMessageW(controls[i], WM_SETFONT, (WPARAM)g_font, TRUE);
    SendMessageW(g_source, WM_SETFONT, (WPARAM)g_mono, TRUE);
}

static const int kColWidth[] = { 240, 50, 260, 70 };  // Name includes the thumbnail

static void Layout(void) {
    RECT rc;
    GetClientRect(g_main, &rc);
    SendMessageW(g_status, WM_SIZE, 0, 0);
    RECT sr;
    GetWindowRect(g_status, &sr);
    int sb = sr.bottom - sr.top;
    int parts[] = { Px(160), -1 };
    SendMessageW(g_status, SB_SETPARTS, 2, (LPARAM)parts);

    int pad = Px(6), row = Px(26), btn = Px(84), filt = Px(200);
    int x = pad, w = rc.right;
    MoveWindow(g_path, x, pad, w - 4 * pad - 2 * btn - filt, row, TRUE);
    MoveWindow(g_browse, w - 2 * pad - 2 * btn - filt - pad, pad, btn, row, TRUE);
    MoveWindow(g_rescan, w - pad - btn - filt - pad, pad, btn, row, TRUE);
    MoveWindow(g_filter, w - pad - filt, pad, filt, row, TRUE);
    int top = row + 2 * pad, h = rc.bottom - top - sb;
    int listw = w * 2 / 5;
    MoveWindow(g_list, 0, top, listw, h, TRUE);
    MoveWindow(g_view, listw + pad / 2, top, w - listw - pad / 2, h, TRUE);
    MoveWindow(g_source, listw + pad / 2, top, w - listw - pad / 2, h, TRUE);
}

static void Create(HWND h) {
    g_main = h;
    HINSTANCE inst = GetModuleHandleW(NULL);
    g_path = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"", WS_CHILD | WS_VISIBLE | ES_AUTOHSCROLL, 0, 0, 0, 0, h,
                             (HMENU)IDC_PATH, inst, NULL);
    SendMessageW(g_path, EM_SETCUEBANNER, TRUE, (LPARAM)L"Folder of models (type a path and press Enter, or drop a folder here)");
    g_browse = CreateWindowW(L"BUTTON", L"Browse...", WS_CHILD | WS_VISIBLE, 0, 0, 0, 0, h, (HMENU)IDC_BROWSE, inst, NULL);
    g_rescan = CreateWindowW(L"BUTTON", L"Rescan", WS_CHILD | WS_VISIBLE, 0, 0, 0, 0, h, (HMENU)IDC_RESCAN, inst, NULL);
    g_filter = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"", WS_CHILD | WS_VISIBLE | ES_AUTOHSCROLL, 0, 0, 0, 0, h,
                               (HMENU)IDC_FILTER, inst, NULL);
    SendMessageW(g_filter, EM_SETCUEBANNER, TRUE, (LPARAM)L"Filter");
    g_list = CreateWindowExW(0, WC_LISTVIEWW, L"", WS_CHILD | WS_VISIBLE | LVS_REPORT | LVS_OWNERDATA |
                             LVS_SINGLESEL | LVS_SHOWSELALWAYS, 0, 0, 0, 0, h, (HMENU)IDC_LIST, inst, NULL);
    ListView_SetExtendedListViewStyle(g_list, LVS_EX_FULLROWSELECT | LVS_EX_DOUBLEBUFFER);
    SetWindowTheme(g_list, L"Explorer", NULL);
    static const wchar_t* const cols[] = { L"Name", L"Type", L"Path", L"Size" };
    for (int i = 0; i < 4; i++) {
        LVCOLUMNW c = { LVCF_TEXT | LVCF_WIDTH | LVCF_FMT, i == 3 ? LVCFMT_RIGHT : LVCFMT_LEFT, Px(kColWidth[i]),
                        (wchar_t*)cols[i] };
        ListView_InsertColumn(g_list, i, &c);
    }
    g_view = CreateWindowW(L"OpticsView", L"", WS_CHILD | WS_VISIBLE | WS_CLIPSIBLINGS | WS_CLIPCHILDREN, 0, 0, 0, 0,
                           h, NULL, inst, NULL);
    g_source = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"", WS_CHILD | ES_MULTILINE | ES_READONLY | WS_VSCROLL |
                               WS_HSCROLL | ES_AUTOVSCROLL | ES_AUTOHSCROLL, 0, 0, 0, 0, h, (HMENU)IDC_SOURCE, inst, NULL);
    g_status = CreateWindowW(STATUSCLASSNAMEW, L"", WS_CHILD | WS_VISIBLE | SBARS_SIZEGRIP, 0, 0, 0, 0, h,
                             (HMENU)IDC_STATUS, inst, NULL);
    MakeFonts();
    StartThumbs();
    DragAcceptFiles(h, TRUE);
    SetStatus(1, "Drag a model to orbit, scroll to zoom, double-click to reset. Enter on a file opens it.");
}

static LRESULT CALLBACK MainProc(HWND h, UINT msg, WPARAM wp, LPARAM lp) {
    switch (msg) {
    case WM_CREATE:
        Create(h);
        return 0;
    case WM_SIZE:
        Layout();
        return 0;
    case WM_DPICHANGED: {
        RECT* r = (RECT*)lp;
        SetWindowPos(h, NULL, r->left, r->top, r->right - r->left, r->bottom - r->top, SWP_NOZORDER);
        MakeFonts();
        for (int i = 0; i < 4; i++) ListView_SetColumnWidth(g_list, i, Px(kColWidth[i]));
        MakeThumbList();
        ResetThumbs();  // re-render at the new size as rows are drawn
        Layout();
        return 0;
    }
    case WM_COMMAND:
        if (LOWORD(wp) == IDC_BROWSE) Browse();
        else if (LOWORD(wp) == IDC_RESCAN) StartScan();
        else if (LOWORD(wp) == IDC_FILTER && HIWORD(wp) == EN_CHANGE) Refilter();
        return 0;
    case WM_DROPFILES: {
        wchar_t path[MAX_PATH * 2];
        if (DragQueryFileW((HDROP)wp, 0, path, MAX_PATH * 2)) {
            if (!PathIsDirectoryW(path)) PathRemoveFileSpecW(path);
            SetRoot(path);
        }
        DragFinish((HDROP)wp);
        return 0;
    }
    case WM_TIMER:
        if (g_scanning) {
            char text[64];
            sprintf_s(text, sizeof(text), "Scanning... %ld found", g_scanning->found);
            SetStatus(0, text);
        }
        return 0;
    case WM_APP_SCANNED: {
        Library* lib = (Library*)lp;
        if (lib != g_scanning) {  // superseded by a newer scan
            Library_Free(lib);
            free(lib);
            return 0;
        }
        KillTimer(h, TIMER_PROGRESS);
        g_scanning = NULL;
        DropModel();
        InvalidateRect(g_view, NULL, FALSE);
        if (g_lib) { Library_Free(g_lib); free(g_lib); }
        g_lib = lib;
        ResetThumbs();
        Refilter();
        for (int i = 0; g_select && i < g_nshown; i++)
            if (!g_lib->items[g_shown[i]].inner && _stricmp(g_lib->items[g_shown[i]].path, g_select) == 0) {
                ListView_SetItemState(g_list, i, LVIS_SELECTED | LVIS_FOCUSED, LVIS_SELECTED | LVIS_FOCUSED);
                ListView_EnsureVisible(g_list, i, FALSE);
            }
        free(g_select);
        g_select = NULL;
        return 0;
    }
    case WM_APP_THUMB:
        ThumbDone((ThumbJob*)lp);
        return 0;
    case WM_NOTIFY: {
        NMHDR* n = (NMHDR*)lp;
        if (n->hwndFrom != g_list) break;
        if (n->code == LVN_GETDISPINFOW) {
            NMLVDISPINFOW* d = (NMLVDISPINFOW*)lp;
            if (d->item.mask & LVIF_TEXT) GetCell(d->item.iItem, d->item.iSubItem, d->item.pszText, d->item.cchTextMax);
            if ((d->item.mask & LVIF_IMAGE) && d->item.iItem >= 0 && d->item.iItem < g_nshown)
                d->item.iImage = ThumbFor(g_shown[d->item.iItem]);
        } else if (n->code == LVN_ITEMCHANGED) {
            NMLISTVIEW* v = (NMLISTVIEW*)lp;
            if ((v->uNewState & LVIS_SELECTED) && !(v->uOldState & LVIS_SELECTED) && v->iItem >= 0 && v->iItem < g_nshown)
                ShowItem(&g_lib->items[g_shown[v->iItem]]);
        } else if (n->code == LVN_ITEMACTIVATE) {
            NMITEMACTIVATE* a = (NMITEMACTIVATE*)lp;
            if (a->iItem >= 0 && a->iItem < g_nshown) {
                wchar_t* w = Widen(g_lib->items[g_shown[a->iItem]].path);  // a member opens its zip
                if (w) ShellExecuteW(h, NULL, w, NULL, NULL, SW_SHOWNORMAL);
                free(w);
            }
        } else if (n->code == LVN_COLUMNCLICK) {
            int col = ((NMLISTVIEW*)lp)->iSubItem;
            g_sortdesc = col == g_sortcol ? !g_sortdesc : 0;
            g_sortcol = col;
            Refilter();
        }
        return 0;
    }
    case WM_DESTROY:
        PostQuitMessage(0);
        return 0;
    }
    return DefWindowProcW(h, msg, wp, lp);
}

int WINAPI wWinMain(HINSTANCE inst, HINSTANCE prev, PWSTR cmd, int show) {
    (void)prev;
    (void)cmd;
    CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    INITCOMMONCONTROLSEX icc = { sizeof(icc), ICC_LISTVIEW_CLASSES | ICC_BAR_CLASSES };
    InitCommonControlsEx(&icc);

    WNDCLASSW vc = { CS_OWNDC | CS_DBLCLKS, ViewProc, 0, 0, inst, NULL, LoadCursor(NULL, IDC_SIZEALL), NULL, NULL,
                     L"OpticsView" };
    RegisterClassW(&vc);
    WNDCLASSW mc = { 0, MainProc, 0, 0, inst, LoadIconW(inst, MAKEINTRESOURCEW(1)), LoadCursor(NULL, IDC_ARROW),
                     (HBRUSH)(COLOR_BTNFACE + 1), NULL, L"OpticsMain" };
    RegisterClassW(&mc);
    UINT dpi = GetDpiForSystem();
    HWND h = CreateWindowExW(0, L"OpticsMain", L"Optics", WS_OVERLAPPEDWINDOW, CW_USEDEFAULT, CW_USEDEFAULT,
                             MulDiv(1280, dpi, 96), MulDiv(800, dpi, 96), NULL, NULL, inst, NULL);
    if (!h) return 1;
    ShowWindow(h, show);
    SetFocus(g_list);

    // A folder on the command line wins, or a file (its folder, with it
    // selected: "Open with"); otherwise the last folder used.
    int argc;
    wchar_t** argv = CommandLineToArgvW(GetCommandLineW(), &argc);
    wchar_t last[MAX_PATH * 2], full[MAX_PATH * 2];
    DWORD size = sizeof(last);
    if (argc > 1 && GetFullPathNameW(argv[1], MAX_PATH * 2, full, NULL)) {
        if (!PathIsDirectoryW(full)) {
            g_select = Narrow(full);
            PathRemoveFileSpecW(full);
        }
        SetRoot(full);
    }
    else if (RegGetValueW(HKEY_CURRENT_USER, L"Software\\Optics", L"LastPath", RRF_RT_REG_SZ, NULL, last, &size) == ERROR_SUCCESS)
        SetRoot(last);
    LocalFree(argv);

    MSG m;
    while (GetMessageW(&m, NULL, 0, 0) > 0) {
        // Enter in the path box rescans; Enter in the list opens the file.
        if (m.message == WM_KEYDOWN && m.wParam == VK_RETURN && m.hwnd == g_path) {
            StartScan();
            continue;
        }
        TranslateMessage(&m);
        DispatchMessageW(&m);
    }
    return 0;
}
