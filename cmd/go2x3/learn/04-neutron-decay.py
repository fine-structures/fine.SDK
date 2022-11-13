from py2x3 import *


def sumTraces(Nv, a, b):
    Ta = a.Traces(Nv)
    Tb = b.Traces(Nv)
    Tsum = Nv * [0]
    for i in range(Nv):
        Tsum[i] = Ta[i] + Tb[i]
    return Tsum
    
def diffTraces(Nv, a, b):
    Ta = a.Traces(Nv)
    Tb = b.Traces(Nv)
    Tsum = Nv * [0]
    for i in range(Nv):
        Tsum[i] = Ta[i] - Tb[i]
    return Tsum
    


n0     = Graph("1-2-3-1-4")
proton = Graph("1-2-3")
qdq    = Graph("1^-2-3^")
W      = Graph("1^^") # W- (charged weak)
e      = Graph("1")
gamma  = Graph("1---2")
anti_v = Graph("1^^-2^^")

print('''
---------------------------------------------------------
Beta Decay:

The actors:''')
n0.Print("neutron",     codes=True, traces=5).Go()
proton.Print("proton",  codes=True, traces=5).Go()
W.Print("W-",           codes=True, traces=5).Go()
gamma.Print("gamma",    codes=True, traces=5).Go()
e.Print("e",            codes=True, traces=5).Go()
anti_v.Print("~ve",     codes=True, traces=5).Go()

print('''
Commonly known n0 beta decay products (note equal n0 Traces):''')



W_proton = Graph(W, proton)
print("W- +  p:  ", W_proton)

e_qdq = Graph(e, qdq)
print("e- + qdq: ", e_qdq)

print('''
Let's verify that the Traces all add up as expected:''')

print("sum(W- + p):  ", sumTraces(4, W, proton))
print("sum(e + qdq): ", sumTraces(4, e, qdq))


print('''
So, does a neutron only have 2 decay modes??
We can use PrimeModes() to analyze a given Traces set and performs a particle "prime" factorization, 
yielding all possible sets of primes that produce the Traces in question:''')
n0.PrimeModes().Print("n0 prime mode").Go()

print('''...so yes, there are only 2 decay modes.''')

print('''
What if a photon interacts with a neutron??''')
n0_gamma = Graph(n0, "1---2")
print("n0 + γ :", n0_gamma)

print('''
Let's use PrimeModes() to see what this v=6 system now factors into.
We should see *at least* the above 2 (with an additional e ~e pair from the gamma)...''')
n0_gamma.PrimeModes().Print("n0 + γ prime mode").Go()

print('''
Behold, there is a THIRD prime factorization!
It turns out this particle prime is of the few v=3 primes that only has forms with one or more negative edges.
We can verify this by adding all the Traces up in any order...''')
e_e_W_ydy1 = Graph(e, e, W, "1^-2=3~1")
print("e + e + W- + y~dy: ", e_e_W_ydy1)
e_e_W_ydy2 = Graph("1^-2=3~1", e, W, e)
print("y~dy + e + W- + e: ", e_e_W_ydy2)
e_e_W_ydy3 = Graph(W, e, "1^-2=3~1", e)
print("W- + e + y~dy + e: ", e_e_W_ydy3)

print('''
Note how W- + e- => anti-neutrino, implying that a gamma is consumed during neutron decay:''')
anti_v.Print("~ve ", traces=8).Go()
Graph("1^^; 1^^^").Print("W- + e", traces=8).Go()
#antineutreno.PrimeModes().Print("~ve PRIMES", None, 10).Go() # This can be used instead to demonstrate that a ~ve is composed of W- + e


print('''
So how often (what relative ratios) can we expect to see for a n0 + gamma interaction?
We use PhaseModes() to calculate the number of equivalent forms of a given particle:''')
ydy = Graph("1^-2=3~1")
N_ydy = ydy   .PhaseModes().Print("y~dy phase mode").Go()
print()
N_qdq = qdq   .PhaseModes().Print("qdq phase mode").Go()
print()
N_p   = proton.PhaseModes().Print("proton phase mode").Go()

print('''
Number of phase modes for each possible n0+γ product:
    y~dy:   %d
    qdq:    %d
    proton: %d
    
And since other particles in the products are v=1, they only have one possible mode, meaning
the above values predict the ratios of the 3 possible n0+γ products.
''' % (N_ydy, N_qdq, N_p))