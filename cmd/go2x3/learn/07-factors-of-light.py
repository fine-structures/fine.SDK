
from py2x3 import *

print("\n")

printOpts = {
    'graph':  True,
    'matrix': True,
    'codes':  True,
}


light = [
    ["gamma", "1---2"],
    ["y4",    "1=2-3=4-1"],
    ["y6",    "1=2-3=4-5=6-1"],
    ["y8",    "1=2-3=4-5=6-7=8-1"],
    ["higgs", "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"],
]


for Xname, graphStr in light:
    
    X = Graph(graphStr)
    print("\n  ===   %s phases  === \n" % Xname)
    X.PhaseModes().Print(Xname + " phase", **printOpts).Go()
    
    print("\n  ===   %s PRIMES  === \n" % Xname)
    X.PrimeModes().Print(Xname + " PRIME", **printOpts).Go()