
from py2x3 import *

print("\n")

verbose = {
    'graph':  True,
    'matrix': True,
    'codes':  True,
    'cycles': True,
    'traces': 8,
}

basic = {
    'codes':  True,
}

p = Graph("1")
p.PhaseModes().Print("e- (electron)", **verbose).Go()

p = Graph("1^^^")
p.PhaseModes().Print("~e (positron)", **verbose).Go()

p = Graph("1---2")
p.PhaseModes().Print("γ0 (photon)", **verbose).Go()

print("\n")

p = Graph("1^")
p.PhaseModes().Print("W+ (charged weak)", **verbose).Go()

p = Graph("1^^")
p.PhaseModes().Print("W- (charged weak)", **verbose).Go()

p = Graph("1^-2^")
p.PhaseModes().Print("Z0 (neutral weak)", **verbose).Go()


print("\n")

p = Graph("1")
p.PhaseModes().Print("e-", **verbose).Go()

p = Graph("1-2-3")
p.PhaseModes().Print("p+", **verbose).Go()

n = Graph("1-2-3-4-2")
n.PhaseModes().Print("n0", **verbose).Go()

print("\n")

tetra = Graph("1-2-3-1-4-2, 4-3")
tetra.PhaseModes().Print("tetra", **basic).Go()

print("\n")

p = Graph("1")
p.PhaseModes().Print("e- (electron)", **basic).Go()

p = Graph("1-2--3")
p.PhaseModes().Print("µ- (muon)    ", **basic).Go()

p = Graph("1-2--3-4--5")
p.PhaseModes().Print("τ  (tau)     ", **basic).Go()

print("\n")

K4 = Graph("1-2-3^-4-1, 2-4")
K4.PhaseModes().Print("K4", **verbose).Go()

print("\n")
Graph("1^=2-3-~4").PhaseModes().Print("tricky1", **verbose).Go()

print("\n")
Graph("1^-~2-3-~4").PhaseModes().Print("tricky2", **verbose).Go()


print("\n")

K8 = Graph("1^-2-3-4-5^-6-7-8-1, 2-8, 4-6")
K8.PhaseModes().Print("K8", **basic).Go()

print("\n")

higgs = Graph("1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8")
higgs.PhaseModes().Print("higgs", **basic).Go()

print("\n")

d4 = Graph("1-2-3-4-1")
d4.PhaseModes().Print("d4", **basic).Go()

print("\n")

d5 = Graph("1-2-3-4-5-1")
d5.PhaseModes().Print("d5", **basic).Go()

print("\n")

d6 = Graph("1-2-3-4-5-6-1")
d6.PhaseModes().Print("d6", **basic).Go()


print("\n")

y4 = Graph("1-2=3-4=1")
y4.PhaseModes().Print("γ4", **basic).Go()

print("\n")

y6 = Graph("1-2=3-4=5-6=1")
y6.PhaseModes().Print("γ6", **basic).Go()

print("\n")

y8 = Graph("1-2=3-4=5-6=7-8=1")
y8.PhaseModes().Print("γ8", **basic).Go()



