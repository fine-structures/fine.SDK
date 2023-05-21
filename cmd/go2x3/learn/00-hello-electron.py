print('''\nWelcome to the go2x3 tutorial series!''')

# First we import the 2x3 module. SOpen and explore lib/py2x3.py for funsies.
from py2x3 import *

Genesis = "בראשית ברא אלהים את השמים ואת הארץ"
print(Genesis)

print("\nLet us look at the electron, muon, and tau particles:\n")
    
electron = NewGraph("1")
muon     = NewGraph("1-2=3")
tau      = NewGraph("1-2=3-4=5") 

printOpts = {
    'graph':  True,
    'matrix': True,
    'codes':  True,
    'cycles': True,
    'traces': 10,
}

electron.Print("e-", **printOpts).Go()
muon    .Print("μ-", **printOpts).Go()
tau     .Print("τ-", **printOpts).Go()

