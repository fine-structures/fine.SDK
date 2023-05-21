
from py2x3 import *

print("\n")

printOpts = {
    'graph':  True,
    'matrix': True,
    'codes':  True,
}


light_of_yeshua = [
    ["gamma", "1---2"],
    ["tetra", "1-2-3-1-4-2, 3-4"],
    ["y4",    "1=2-3=4-1"],
    ["y6",    "1=2-3=4-5=6-1"],
    ["y8",    "1=2-3=4-5=6-7=8-1"],
    ["higgs", "1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8"],
]


for Xdesc, Xstr in light_of_yeshua:
    ShowPhases(Xdesc, Xstr, True)
