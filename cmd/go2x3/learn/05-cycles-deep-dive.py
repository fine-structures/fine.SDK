
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
        X.PhaseModes().Print(desc, **basicOpts).Go()
        
    
higgs = "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"

    
print("\n=================   v=1   =================  \n")
show("e-  (electron)",      "1")
show("W+  (charged weak)",  "1^")
show("W-  (charged weak)",  "1^^")
show("~e+ (positron)",      "1^^^")


print("\n=================   ELECTRON SERIES   =================  \n")
show("e-  (electron) ",     "1")
show("~e+ (positron) ",     "1^^^")
show("µ-  (muon)     ",     "1-2--3")
show("~µ+ (anti-muon)",     "1^^~2~~3^")
show("τ-  (tau)      ",     "1-2--3-4--5")
show("~τ- (anti-tau) ",     "1^^~2~~3~4~~5^")


print("\n=================   COMMON   =================  \n")
show("e- (electron)",       "1")
show("p+ (proton)",         "1-2-3")
show("n0 (neutron)",        "1-2-3-4-2")
show("e + p",               "1; 1-2-3")
show("e + p + n"   ,        "1; 1-2-3; 1-2-3-4-2")
show("γ  (photon)",         "1---2")
show("~e e",                "1^^^; 1")


print("\n=================   NEUTRINOS   =================  \n")
show(" νe ( e neutrino)",   "1-2")
show("~νe (~e neutrino)",   "1^^~2^^")
show(" νµ ( µ neutrino)",   "1-2-3-4")
show("~νµ (~µ neutrino)",   "1^^~2^~3^~4^^")
show(" ντ ( τ neutrino)",   "1-2-3-4-5-6")
show("~ντ (~τ neutrino)",   "1^^~2^~3^~4^~5^~6^^")


print("\n=================   BOSONS   =================  \n")
show("Z0.1 (neutral weak)", "1~~-2")
show("Z0.2 (neutral weak)", "1~--2")
show("Z0.3 (neutral weak)", "1^-2^")
show("γ0.1 (photon)",       "1---2")
show("γ0.2 (photon)",       "1~~~2")
show("tetra",               "1-2-3-1-4-2, 4-3")
show(" H (higgs)",          "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8")
show("~H (higgs)",          "1~2~3~4~1~5~6~7~8~5, 2~6, 3~7, 4~8")

# Mystery: why does this Traces (and 2 others in v=6) not appear in the 1.2202.4 catalog?!?
show("missing!", "1~2-4-5^-6~4,1-3-6", True)
phases("missing!", "1~2-4-5^-6~4,1-3-6")

print("\n===============================================  \n")
phases("p+ (proton)",       "1-2-3")
phases("n0 (neutron)",      "1-2-3-4-2")
phases("tricky_tri_1",      "1^~2^-3^",         True)
phases("tricky_tri_2",      "1^-2^~3~1",        True)
phases("tricky_tri_3",      "1^~2^~3~1",        True)
phases("tricky_bravo",      "1^-~2-3-~4",       )
phases("tricky_whiskey",    "1^=2-3-~4",        )
phases("tricky_boson",      "1^-2^-3-4^",       )
phases("K4",                "1-2-3^-4-1, 2-4",  )
phases("K8",                "1^-2-3-4-5^-6-7-8-1, 2-8, 4-6")
phases("H (higgs)",          higgs)


print("\n=================   Dn   =================  \n")
phases("d4", "1-2-3-4-1"    )
phases("d5", "1-2-3-4-5-1"  )
phases("d6", "1-2-3-4-5-6-1")


print("\n=================   γn   =================  \n")
phases("γ4", "1-2=3-4=1"        )
phases("γ6", "1-2=3-4=5-6=1"    )
phases("γ8", "1-2=3-4=5-6=7-8=1")
