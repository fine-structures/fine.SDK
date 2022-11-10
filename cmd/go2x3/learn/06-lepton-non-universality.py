from py2x3 import *

muon  = Graph("1-2=3")
amuon = Graph("1^^-2=3^")
K8    = Graph("1^-2-3-4-5^-6-7-8-1, 2-8, 4-6")
K4    = Graph("1-2-3^-4-1, 2-4")
K4n1  = Graph("1~2-3^-4-1, 2-4")
qyd   = Graph("1^-2=3")
v6    = Graph("1=2-3-4-5=6")
y6    = Graph("1-2=3-4=5-6=1")
K8y   = Graph("1", "1^^^", K8)

print('''
The following explores quantifying the products (probabilities) of decay products of
    e + ~e + K8
    
We first confirm other forms the above by comparing Traces of different particle combos:''')
print("    e~e + K8:             ", K8y                                 .Traces())
print("    e~e + K4~1 + K4:      ", Graph("1", "1^^^", K4n1, K4        ).Traces())
print("    e~e + K4~1 + π + qyd: ", Graph("1", "1^^^", K4n1, "1^^", qyd).Traces())
print("    π + ~µ + v6:          ", Graph("1^^", amuon, v6             ).Traces())
print("    π + ~µ + µ + qyd:     ", Graph("1^^", amuon, muon,    qyd   ).Traces())
print("    K4      + y6:         ", Graph(K4, y6                       ).Traces())
print("    K4      + µ~µ:        ", Graph(K4, muon, amuon              ).Traces())

print()
K4.PrimeModes().Print("K4 PRIME").Go()
K4_phases = K4.PhaseModes().Print("K4 phase").Go()

print()
K4n1.PrimeModes().Print("K4~1 PRIME").Go()
K4n1_phases = K4n1.PhaseModes().Print("K4~1 phase").Go()

print()
muon.PrimeModes().Print(" muon PRIME").Go()
muon_phases = muon.PhaseModes().Print(" muon phase").Go()

print()
amuon.PrimeModes().Print("~muon PRIME").Go()
amuon_phases = amuon.PhaseModes().Print("~muon phase").Go()

print()
qyd.PrimeModes().Print("qyd PRIME").Go()
qyd_phases = qyd.PhaseModes().Print("qyd phase").Go()

print()
v6.PrimeModes().Print(" v6 PRIME").Go()
v6_phases = v6.PhaseModes().Print(" v6 phase").Go()

print()
m_am_K4  = muon_phases * amuon_phases * K4_phases
m_am_qyd = muon_phases * amuon_phases * qyd_phases
print("muon(%d) x ~muon(%d) x K4(%d) combos: %d" %(muon_phases, amuon_phases, K4_phases, m_am_K4)) 
print("muon(%d) x ~muon(%d) x qyd(%d) combos: %d" %(muon_phases, amuon_phases, qyd_phases,  m_am_qyd)) 

print("...however, we exclude the latter since the LHCb detectors throw out µ~µ hits unless there is neutral kaon (K4) detected, so:")
total_muon_pp = m_am_K4
print("Total muon + ~muon pair producing modes: %d" % (total_muon_pp))

print()
y6.PrimeModes().Print("y6 PRIME").Go()
y6.PhaseModes().Print("y6 phase").Go()

print("Well, what do you know, the y6 decomposes into (e~e + K4~1) OR (muon + ~muon)")
print('''This means an e~e pair can come from: 
    ~π + qyd + (e~e + K4~1), or
    K4 +       (e~e + K4~1)''')
y6_qm = qyd_phases * 1 * K4n1_phases
print("qyd(%d) x e~e(%d) x K4~1(%d) combos: %d" % (qyd_phases, 1, K4n1_phases, y6_qm))

y6_K4 = K4_phases * 1 * K4n1_phases
print("K4(%d) x e~e(%d) x K4~1(%d) combos: %d" % (K4_phases, 1, K4n1_phases, y6_K4))

print('''
Another decay mode of our v=10 system contains y6 with two neg loops. 
Let's verify that it doesn't have any prime modes or phases that would create lepton pairs:''')
y4_2 = Graph("1-2^-3^-4=5-6=1")
y4_2.PrimeModes().Print("y4_2 PRIME").Go()
y4_2.PhaseModes().Print("y4_2 phase").Go()

print('''
And finally, we know an e~e pair can come from:
    e~e + K8''')
K8.PrimeModes().Print("K8 PRIME").Go()
K8_phases = K8.PhaseModes().Print("K8 phase").Go()

print("e~e(%d) x K8(%d) combos: %d" % (1, K8_phases, K8_phases))

total_e_pp = y6_qm + y6_K4 + K8_phases

pp_ratio = total_muon_pp / total_e_pp
print('''
total_muon_pp(%d) / total_e_pp(%d) => %f''' % (total_muon_pp, total_e_pp, pp_ratio))


print('''
As of January 2022, the LHCb team is measuring .85 (σ≈.05).
For more info -- http://www.sci-news.com/physics/lepton-universality-tests-10189.html
    
''')

# This verifies that there are no unaccounted decay modes of 'reactants' (that would otherwise skew the lepton count)
#K8y.PrimeModes().Print("K8y PRIME").Go()