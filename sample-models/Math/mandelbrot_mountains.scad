

draw_base=0;
draw_name=1;
draw_label=1;
letter_ht=3;
num_letters=10;

total_mm = 100;
resolution_mm = 1 ;
base_mm=1;

max_height_mm=40;
max_iterations=128;

//z_left = -0.7;
//z_bottom = -.5;
//z_length = .08;

z_left =  -2 ;
z_bottom = -1.5;
z_length = 3;


function z_add (z1,z2)=[0,z1[1]+z2[1],z1[2]+z2[2]];

function z_multiply (z1,z2)=[0, z1[1]*z2[1]-z1[2]*z2[2],  z1[1]*z2[2]+z1[2]*z2[1] ];

function z_norm (z) = sqrt(z[1]*z[1]+z[2]*z[2]);

function z_mandel (z,c)=z_add(z_multiply(z,z),c);

function convert_to_z (i,j) = 
[0,
z_left + z_length * i /total_mm,
z_bottom + z_length * j /total_mm ] ;

function mandel_count (z,c,n)= ((z_norm(z)<=2) && (n<=max_iterations)) ? mandel_count(z_mandel(z,c),c,n+1) : n ;

module draw_rectangle (x1,y1,z1,x2,y2,z2) {
    
    Poly_Points = [
        [x1,y1,z1],
        [x2,y1,z1],
        [x2,y2,z1],
        [x1,y2,z1],
        [x1,y1,z2],
        [x2,y1,z2],
        [x2,y2,z2],
        [x1,y2,z2]];
        
    Poly_Faces = [
        [0,1,2,3],  // bottom
        [4,5,1,0],  // front
        [7,6,5,4],  // top
        [5,6,2,1],  // right
        [6,7,3,2],  // back
        [7,4,0,3]]; // left
        
    render ()polyhedron (Poly_Points, Poly_Faces );
}




for (i=[0:resolution_mm:total_mm]) {
    for (j=[0:resolution_mm:total_mm]) {
        
        c_point=convert_to_z(i,j);
        c_count= mandel_count([0,0,0],c_point,0);
        
        draw_rectangle (
            i,j,0,
            i+resolution_mm,
            j+resolution_mm,
            c_count*max_height_mm/max_iterations+base_mm);
               
    };
};


if (draw_label==1) {
    
    draw_rectangle (total_mm,0,0,total_mm+letter_ht*num_letters,(letter_ht+2)*4+1,1);
    

    translate([total_mm+letter_ht,(letter_ht+2)*3+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("x= ",z_left), font = "Liberation Sans", size=letter_ht);
        }
    }
    translate([total_mm+letter_ht,(letter_ht+2)*2+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("y= ",z_bottom), font = "Liberation Sans", size=letter_ht);
        }
    }
    translate([total_mm+letter_ht,(letter_ht+2)*1+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("d= ",z_length), font = "Liberation Sans", size=letter_ht);
        }
    }
    translate([total_mm+letter_ht,(letter_ht+2)*0+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("n= ",max_iterations), font = "Liberation Sans", size=letter_ht);
        }
    }
};

if (draw_name==1) {
    draw_rectangle (total_mm,total_mm,0,total_mm+letter_ht*num_letters,total_mm-(letter_ht+2)*4+1,1);
    translate([total_mm+letter_ht/2,total_mm-(letter_ht+1)*2+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("By"), font = "Liberation Sans", size=letter_ht);
        }
    }
    translate([total_mm+letter_ht/2,total_mm-(letter_ht+1)*3+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("Pranab"), font = "Liberation Sans", size=letter_ht);
        }
    }
    translate([total_mm+letter_ht/2,total_mm-(letter_ht+1)*4+1,1]) {
        linear_extrude(height = 2, center = true, convexity = 10, twist = 0, slices = 20, scale = 1.0) {
            text(str("Majumder"), font = "Liberation Sans", size=letter_ht);
        }
    }
}


if (draw_base==1){
    b_st=2;
    b_f=1.4;
    b_r1=1;
    b_b=0;
    
    for (i=[b_st/2:b_st:b_st/2+b_st*floor((total_mm+draw_label*35)/b_st)]){
        for (j=[b_st/2:b_st:b_st/2+b_st*floor((total_mm)/b_st)]){
            render() 
            translate ([i,j,-b_f*b_st/2-2*b_b])
            cylinder(h=b_f*b_st/2-b_r1/2,r1=b_r1,r2=b_f*b_st/2, center=false,$fn=8);
        }
    };


//    
//    if (b_b>0){
//        for (i=[-2*b_b:2*b_b:total_mm+draw_label*letter_ht*num_letters+resolution_mm+b_b]) {
//            for (j=[-2*b_b:2*b_b:total_mm+resolution_mm+b_b]){
//                render()
//                translate ([i,j,-b_b])
//                sphere (r=b_b,$fn=10);
//            }
//        }
//    }
}
   
