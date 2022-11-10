
from py2x3 import *



verboseOpts = {
    'graph':  True,
    'matrix': True,
    'codes':  True,
    'cycles': True,
    'traces': 8,
}

basicOpts = {
    'codes':  True,
}

def show(desc, Xstr):
    X = Graph(Xstr)
    X.Print(desc, **verboseOpts).Go()
    

def phases(desc, Xstr, verbose = False):
    X = Graph(Xstr)
    if verbose:
        X.PhaseModes().Print(desc, **verboseOpts).Go()
    else:
        X.PhaseModes().Print(desc, **basicOpts)
        
    
higgs = "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"

    
print("\n   ======.   V1   ======.  \n")

show("e-  (electron)",      "1")
show("W+  (charged weak)",  "1^")
show("W-  (charged weak)",  "1^^")
show("~e+ (positron)",      "1^^^")
show("γ   (photon)",        "1---2")



print("\n   ======.   ELECTRON SERIES   ======.  \n")

show("e-  (electron) ", "1")
show("µ-  (muon)     ", "1-2--3")
show("~µ+ (anti-muon)", "1^^~2~~3^")
show("τ-  (tau)      ", "1-2--3-4--5")



print("\n   ======.   COMMON   ======.  \n")

show("e- (electron)",   "1")
show("p+ (proton)",     "1-2-3")
show("n0 (neutron)",    "1-2-3-4-2")



print("\n   ======.   'EXOTIC'   ======.  \n")

show("Z0 (neutral weak)",       "1^-2^")
show("ve (electron neutrino)",  "1-2")
show("vµ (muon neutrino)",      "1-2-3-4")
show("vτ (tau neutrino)",       "1-2")



print("\n   ======.   BOSONS   ======.  \n")

show("γ  (photon)", "1---2")
show("tetra_echo",  "1-2-3-1-4-2, 4-3")
show("Higgs", higgs)



print("\n   =========================.  \n")

phases("K4",                "1-2-3^-4-1, 2-4",  True)
phases("tricky_tango",      "1^=2-3-~4",        True)
phases("tricky_whiskey",    "1^-~2-3-~4",       True)
phases("K8",                "1^-2-3-4-5^-6-7-8-1, 2-8, 4-6")
phases("Higgs", higgs)



print("\n   ======.   Dn   ======.  \n")

phases("d4", "1-2-3-4-1"    )
phases("d5", "1-2-3-4-5-1"  )
phases("d6", "1-2-3-4-5-6-1")



print("\n   ======.   γn   ======.  \n")

phases("γ4", "1-2=3-4=1"        )
phases("γ6", "1-2=3-4=5-6=1"    )
phases("γ8", "1-2=3-4=5-6=7-8=1")



