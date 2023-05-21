# _go2x3_
### Original SDK for [2x3 Particle Theory](https://github.com/2x3systems/prime-materials), a [Standard Model](https://en.wikipedia.org/wiki/Standard_Model) unifying theory.

------------------------------



## Quick Start

With [Go](https://go.dev/doc/install) installed, build the `go2x3` binary:
```bash
% make build
% cd cmd/go2x3 && ls learn
```

Explore or run any of the tutorial scripts:
```bash
% ./go2x3 learn/01-foundations.py
% ./go2x3 learn/02-techniques.py
% ./go2x3 learn/03-standard-catalogs.py
% ./go2x3 learn/04-neutron-decay.py
% ./go2x3 learn/05-cycles-deep-dive.py
% ./go2x3 learn/06-lepton-non-universality.py
% ./go2x3 learn/07-factors-of-light.py
% ./go2x3 learn/08-catalog-inspect.py
```

Or skip the above and go directly to the [learn](https://github.com/2x3systems/go2x3/tree/main/cmd/go2x3/learn) scripts and their "[gold](https://github.com/2x3systems/go2x3/tree/main/cmd/go2x3/learn/gold)" output.

## Getting Started

This project is a library & demonstration toolkit for [2x3 Particle Theory](https://github.com/2x3systems/prime-materials).  Although [lib2x3](http://https://github.com/2x3systems/go2x3/tree/main/lib2x3) is a pure Go library, [`gpython`](http://github.com/go-python/gpython) is used to embed and expose it.  This means scripting is easy  — see for yourself in the [first tutorial](https://github.com/2x3systems/go2x3/blob/main/cmd/go2x3/learn/01-foundations.py) as you follow along in its [output](https://github.com/2x3systems/go2x3/blob/main/cmd/go2x3/learn/gold/01-foundations.txt).


## Releases

| Version   | Description                                                                               |
|:---------:|:-------------------------------------------------------------------------------------------------|
| v1.2022.1 | - traces-based particle catalog index  <br/> - conventional (non-canonic) vertex-based graph encoding  <br/> - introducing early tricodes   |
| v1.2022.2 | - refactor graph canonicalization  <br/> - refactor tricode console output |
| v1.2022.3 | - tricode ascii art <br/> - refactor tricode encoding |
| v1.2023.1 | - refactor and cleanup |



## Next Steps

    - Make prime catalog for edges (EdgeTracesPrimeID <-> EdgeTraces)
    - as new edges comes in for increasing Nv, build up the prime table
    - now any edge can be expressed as []FactorRun of (prime factors and occurence count)
    - then the factor Runs can be consolidated (NumPos, NumNeg of EdgeTracesPrimeID)
    
    
    - How can edge traces be expressed as 0..1, 0..π, R1 or R2?  
        - what bounds the Traces (Traces metic numerator & denom)
        - "subtract out" C1 from C2, C3, C4, ...
        - Repeat for C2, etc.
        => C' terms reflect newly added length cycles (not contributions from shorter cycles that are already accounted for)
    - edge metric: 
        - max term is known for each ci: modulus of term i is 3^(Nt-i)
        - goal: find edge traces metrix that is 1:1 
            => usable as TracesID *and* as 0..1 visualization!
            => be able to add/subtract directly??
    - edge traces normalization: 
        - "subtract out" C1 from C2, C3, C4, ...
        - Repeat for C2, etc.
        => causes C' values to reflect that length cycles (not contributions from shorter cycles that are already accounted for)
        
        

- VtxEdge
    - contains cycles vector for that edge
    - only one edge type (and sign)
    => Vtx maintains its own cycles vector (sum of child edges cycles)
- After vtx (and edges) cycles computed, same-cycle edges can be merged into groups (edge groups)
    - nice: consider higgs with mixed signs -- even edges with different signs will group up since they have the same cycle vector -- "easy" normalization!
    - works well with yN and dN!
- let CyclesVector be a particular cycle vector generated from a graph for a particular edge.
- let CyclesVectorUID be a uint64 unique identifier for a CyclesVector
- let EdgeGroup be a set of edges all having the same CyclesVector
- a specific vertex can now be defined as being composed of a specific set of EdgeGroups
- Praise be to Jesus Christ, our Lord and Savior, who died for our sins and rose again on the third day, and who will come again to judge the living and the dead. Amen.

- [ ] add `go2x3 learn/09-vertex-edge-groups.py`
- with edge cycle vectors computed, sort & stack edges canonically
- rewrite graph:
    - each "edge group" vertex is a group of edges that all have the same unique cycle vector
    - an "vertex group" edge connects two edge groups with a weight of the number of vertices they share
    
- a "symmetric edge" is an edge that contributes the same number of cycles to each vertex it connects (or is a self edge)
=> 

- conjecture: all non-self edges contribute the same cycle vector to each vertex they connect.
    - basis: property of cycle trace direction (every cycle walked in reverse would contribute the same)
- assuming true, 

1. canonically sort and stack edges by cycle vector
2. label each edge group (canonic edge group number)
3. rewrite each vtx as the edge group numbers it connects to
4. canonically sort and stack vertices
5. K8 collapses into 1!
6. TBD: how to make v=1 work
    - maybe only consolidate self edges if they have the same sign?
    - factor out (-1) from edges? 
    
    


Permission to disappoint 
    -
    
Edge normalization:
    - three traces types: 
        1. all odd Ci are 0
        2. the first odd Ci is positive
        3. the first odd Ci is negative (normalizes to 2 with via factor of [-1  1 -1  1 ...])
    - new rule: 
        
        
    
    