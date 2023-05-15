
from py2x3 import *


higgs = "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"
antiH = "1~2~3~4~1~5~6~7~8~5, 2~6, 3~7, 4~8"
    
print("\n=================   v=1   =================  \n")
ShowGraph("e-  (electron)",      "1")
ShowGraph("W+  (charged weak)",  "1^")
ShowGraph("W-  (charged weak)",  "1^^")
ShowGraph("~e+ (positron)",      "1^^^")


print("\n=================   ELECTRON SERIES   =================  \n")
ShowGraph("e-  (electron) ",     "1")
ShowGraph("~e+ (positron) ",     "1^^^")
ShowGraph("µ-  (muon)     ",     "1-2--3")
ShowGraph("~µ+ (anti-muon)",     "1^^~2~~3^")
ShowGraph("τ-  (tau)      ",     "1-2--3-4--5")
ShowGraph("~τ- (anti-tau) ",     "1^^~2~~3~4~~5^")


print("\n=================   COMMON   =================  \n")
ShowGraph("e- (electron)",       "1")
ShowGraph("p+ (proton)",         "1-2-3")
ShowGraph("n0 (neutron)",        "1-2-3-4-2")
ShowGraph("e + p",               "1; 1-2-3")
ShowGraph("e + p + n"   ,        "1; 1-2-3; 1-2-3-4-2")
ShowGraph("γ  (photon)",         "1---2")
ShowGraph("~e e",                "1^^^; 1")


print("\n=================   NEUTRINOS   =================  \n")
ShowPhases(" νe ( e neutrino)",   "1-2")
ShowPhases("~νe (~e neutrino)",   "1^^~2^^")
ShowPhases(" νµ ( µ neutrino)",   "1-2-3-4")
ShowPhases("~νµ (~µ neutrino)",   "1^^~2^~3^~4^^")
ShowGraph(" ντ ( τ neutrino)",    "1-2-3-4-5-6")
ShowGraph("~ντ (~τ neutrino)",    "1^^~2^~3^~4^~5^~6^^")


print("\n=================   BOSONS   =================  \n")
ShowGraph("Z0.1 (neutral weak)", "1~~-2")
ShowGraph("Z0.2 (neutral weak)", "1~--2")
ShowGraph("Z0.3 (neutral weak)", "1^-2^")
ShowGraph("γ0.1 (photon)",       "1---2")
ShowGraph("γ0.2 (photon)",       "1~~~2")
ShowGraph("tetra",               "1-2-3-1-4-2, 4-3")
ShowGraph(" H (higgs)",          higgs)
ShowGraph("~H (higgs)",          antiH)

print("\n===============================================  \n")
ShowPhases("γ  (photon)",       "1---2",            True)
ShowPhases("p+ (proton)",       "1-2-3",            True)
ShowPhases("n0 (neutron)",      "1-2-3-4-2",        True)
ShowPhases("tricky_tri_1",      "1^~2^-3^",         True)
ShowPhases("tricky_tri_2",      "1^-2^~3~1",        True)
ShowPhases("tricky_tri_3",      "1^~2^~3~1",        True)
ShowPhases("tricky_bravo",      "1^-~2-3-~4",       )
ShowPhases("tricky_whiskey",    "1^=2-3-~4",        )
ShowPhases("tricky_boson",      "1^-2^-3-4^",       )
ShowPhases("E0 <=> C1",         "1^-2-3-4, 3-5-6^"  )
ShowPhases("τ-  (tau)      ",   "1-2--3-4--5",      )
ShowPhases("~τ- (anti-tau) ",   "1^^~2~~3~4~~5^",   )
ShowPhases("K4",                "1-2-3^-4-1, 2-4",  )
ShowPhases("K8",                "1^-2-3-4-5^-6-7-8-1, 2-8, 4-6")
ShowPhases("H (higgs)",          higgs)


print("\n=================   Dn   =================  \n")
ShowPhases("d4", "1-2-3-4-1"    )
ShowPhases("d5", "1-2-3-4-5-1"  )
ShowPhases("d6", "1-2-3-4-5-6-1")


print("\n=================   γn   =================  \n")
ShowPhases("γ4", "1-2=3-4=1"        )
ShowPhases("γ6", "1-2=3-4=5-6=1"    )
ShowPhases("γ8", "1-2=3-4=5-6=7-8=1")
