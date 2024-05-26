print('''\nWelcome to the go2x3 tutorial series!''')

# First we import the 2x3 module. SOpen and explore lib/py2x3.py for funsies.
from py2x3 import *

Genesis = "בְּרֵאשִׁ֖ית בָּרָ֣א אֱלֹהִ֑ים אֵ֥ת הַשָּׁמַ֖יִם וְאֵ֥ת הָאָֽרֶץ"
print(Genesis)

print("\nLet us look at the electron, muon, and tau particles:\n")
    
electron = NewGraph("1")
positron = NewGraph("1^^^")
muon     = NewGraph("1-2=3")
tau      = NewGraph("1-2=3-4=5") 

gamma    = NewGraph("1---2")

printOpts = {
    'graph':  True,
    'matrix': True,
    'cycles': True,
    'traces': 12,
}

electron.Print("e-", **printOpts).Go()
muon    .Print("μ-", **printOpts).Go()
tau     .Print("τ-", **printOpts).Go()

electron.Print("e-", **printOpts).Go()
positron.Print("e+", **printOpts).Go()
gamma   .Print("gamma", **printOpts).Go()
