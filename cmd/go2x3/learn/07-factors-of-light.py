
from py2x3 import *

print("\n")

printOpts = {
    'graph':  True,
    'matrix': True,
    'codes':  True,
}


light_of_yashua = [
    ["gamma", "1---2"],
    ["tetra", "1-2-3-1-4-2, 3-4"],
    ["y4",    "1=2-3=4-1"],
    ["y6",    "1=2-3=4-5=6-1"],
    ["y8",    "1=2-3=4-5=6-7=8-1"],
    ["higgs", "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"],
]


for Xname, graphStr in light_of_yashua:
    X = Graph(graphStr)
    print("\n  ===   %s phases  === \n" % Xname)
    X.PhaseModes().Print(Xname + " phase", **printOpts).Go()
    
    print("\n  ===   %s prime factors  === \n" % Xname)
    X.PrimeModes().Print(Xname + " prime factors", **printOpts).Go()
