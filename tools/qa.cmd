@echo off
rem netlab QA (recipe projects/optics.env): %1 is Optics.exe; optics-check.exe sits beside it.
rem Run from the checkout, or on a machine where the recipe's RUN_FILES were copied.
"%~dp1optics-check.exe" --selftest || exit /b 1
"%~dp1optics-check.exe" --check sample-models tests/expected.tsv || exit /b 1
