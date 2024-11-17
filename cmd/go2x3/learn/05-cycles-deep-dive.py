
from py2x3 import *


higgs = "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"
antiH = "1~2~3~4~1~5~6~7~8~5, 2~6, 3~7, 4~8"
    
print("\n=================   COMMON   =================  \n")
ShowGraph(" e- (electron)",      "1")
ShowGraph("~e+ (positron)",      "1^^^")
ShowGraph(" ?+ (unknown+)",      "1^")
ShowGraph(" ?- (unknown-)",      "1^^")
ShowGraph(" γ (photon)",         "1---2")
ShowGraph("~γ (photon)",         "1~~~2")
ShowGraph("~e e (γ scattering",  "1^^^; 1")
ShowGraph("p+ (proton)",         "1-2-3")
ShowGraph("n0 (neutron)",        "1-2-3-4-2")
ShowGraph(" νe (e neutrino)",      "1-2")
ShowGraph("~νe (e anti-neutrino)", "1^^~2^^")
ShowGraph(" νe  ~νe",              "1-2; 1^^~2^^")
ShowGraph("e- p+    (H  atom)",    "1; 1-2-3")
ShowGraph("e- p+ n0 (H2 atom)",    "1; 1-2-3; 1-2-3-4-2")
ShowGraph("   p+ n0 (H2 nucl)",    "1-2-3; 1-2-3-4-2")
ShowGraph("p+ ~p-",                "1-2-3; 1^^~2^~3^^")





print("\n=================   ELECTRON SERIES   =================  \n")
ShowGraph("e-  (electron) ",     "1")
ShowGraph("µ-  (muon)     ",     "1-2--3")
ShowGraph("τ-  (tau)      ",     "1-2--3-4--5")
ShowGraph("Τ-  (super-tau)",     "1-2--3-4--5-6--7")
ShowGraph("Τ-  (mega-tau)",      "1-2--3-4--5-6--7-8--9")
ShowGraph("Τ-  (giga-tau)",      "1-2--3-4--5-6--7-8--9-10--11")
ShowGraph("~e+ (positron) ",     "1^^^")
ShowGraph("~µ+ (anti-muon)",     "1^^~2~~3^")
ShowGraph("~τ- (anti-tau) ",     "1^^~2~~3~4~~5^")




print("\n=================   NEUTRINOS   =================  \n")
ShowGraph(" νe ( e neutrino)",   "1-2")
ShowGraph(" νµ ( µ neutrino)",   "1-2-3-4")
ShowGraph(" ντ ( τ neutrino)",   "1-2-3-4-5-6")
ShowGraph(" ν8 (super neutrino)","1-2-3-4-5-6-7-8")
ShowGraph("~νe (~e neutrino)",   "1^^~2^^")
ShowGraph("~νµ (~µ neutrino)",   "1^^~2^~3^~4^^")
ShowGraph("~ντ (~τ neutrino)",   "1^^~2^~3^~4^~5^~6^^")


print("\n=================   BOSONS   =================  \n")
ShowGraph("Z neutral weak?",     "1--~2")
ShowGraph(" γ (photon)",         "1---2")
ShowGraph(" H (higgs)",          higgs)
ShowGraph("~H (higgs)",          antiH)
ShowGraph("γ4", "1-2=3-4=1"        )
ShowGraph("γ6", "1-2=3-4=5-6=1"    )
ShowGraph("γ8", "1-2=3-4=5-6=7-8=1")


print("\n=================   EDGE SIMILARITY   =================  \n")
ShowGraph(" νe (e neutrino)",    "1-2")
ShowGraph(" νµ (µ neutrino)",    "1-2-3-4")
ShowGraph(" ντ (τ neutrino)",    "1-2-3-4-5-6")
ShowGraph(" ν8 (super neutrino)","1-2-3-4-5-6-7-8")
ShowGraph("y8  (quad-gamma)",    "1=2-3=4-5=6-7=8-1")
ShowGraph("y8d ",                "1-2-3=4-5=6-7=8-1")
ShowGraph("y8d_flip",            "1-2-3-~4-5=6-7=8-1")

ShowGraph("rando1",              "1^^-2-~3-4-5~~6-4")
ShowGraph("rando1~",             "1~2~-3~4~5--6~4")

ShowGraph("rando2",              "1-2-~3-4-~5^"   )
ShowGraph("rando2~",             "1^^~2~-3~4~-5")

ShowGraph("rando3",              "1~2~3-4^-6-2, 3-5~6")
ShowGraph("rando3~",             "1^^-2-3~4~6~2, 3~5^-6")

ShowGraph("d3", "1-2-3-1")
ShowGraph("d5", "1-2-3-4-5-1")

ShowGraph("d4", "1-2-3-4-1")
ShowGraph("d6", "1-2-3-4-5-6-1")
ShowGraph("d8", "1-2-3-4-5-6-7-8-1")

print("\n===============================================  \n")
ShowPhases("tetra",             "1-2-3-1-4-2, 4-3")
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
ShowPhases("K4",                "1-2-3^-4-1, 2-4",  )
ShowPhases("K8",                "1^-2-3-4-5^-6-7-8-1, 2-8, 4-6")
ShowPhases("H (higgs)",          higgs)
ShowPhases("τ-)",               "1-2=3-4=5")


print("\n=================   Dn   =================  \n")
ShowPhases("d4", "1-2-3-4-1"    )
ShowPhases("d5", "1-2-3-4-5-1"  )
ShowPhases("d6", "1-2-3-4-5-6-1")


