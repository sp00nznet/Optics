
module rast(lr=1,c=0.2,h=0.8){
    //cutouts in top
    hull(){
        translate([0,0,1.5-h])linear_extrude(.05) polygon([[lr*(12),1.5],[lr*5.5,0.5],[lr*5.5,-0.5],[lr*(12),-1.5]]);
        translate([0,0,1.5])linear_extrude(.05) polygon([[lr*(12),1.5+c],[lr*5.5,0.5+c/2],[lr*5.5,-0.5-c/2],[lr*(12),-1.5-c]]);
    }
}

$fn=80;


module mount(){
  translate([0,0,3.6]){
    translate([0,0,-3.6])linear_extrude(2){circle(d=31,center=true);}
    difference()
      {
      union(){ // upper part with brakets
        translate([0,0,-3.5])linear_extrude(5)circle(d=25,center=true);
        intersection() {
          translate([0,0,0.9])cube([35,10,1.2],center=true);
          cylinder(h=30,d=29,center=true);
        }
      }
      rast(1,0.5,.6);
      rast(-1,0.5,.6);
      translate([-16,0,0])cylinder(d=4,h=20,center=true); // circular cutout on edge
      translate([16,0,0])cylinder(d=4,h=20,center=true); // other edge
    }
  }
}

mount();
translate([0,0,1])cube([31,31,2],center=true);