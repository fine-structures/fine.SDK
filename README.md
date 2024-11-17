# Fine Structures SDK
### Official SDK for _[Fine Structures](https://github.com/fine-strucutures/prime-materials)_, _a [Standard Model](https://en.wikipedia.org/wiki/Standard_Model) unifying theory_

------------------------------



## Quick Start

With [Go](https://go.dev/doc/install) installed, build the `go2x3` binary:
```bash
% make build
% cd cmd/go2x3 && ls learn
```

Explore or run any of the tutorial scripts:
```bash
% ./go2x3 learn/00-hello-electron.py
% ./go2x3 learn/01-foundations.py
% ./go2x3 learn/02-techniques.py
% ./go2x3 learn/03-standard-catalogs.py
% ./go2x3 learn/04-neutron-decay.py
% ./go2x3 learn/05-cycles-deep-dive.py
% ./go2x3 learn/06-lepton-non-universality.py
% ./go2x3 learn/07-factors-of-light.py
% ./go2x3 learn/08-catalog-inspect.py
```

Or skip the above and go directly to the [learn](https://github.com/fine-structures/fine.SDK/tree/main/cmd/go2x3/learn) scripts and their "[gold](https://github.com/fine-structures/fine.SDK/tree/main/cmd/go2x3/learn/gold)" output.

## Getting Started

This project is a library & demonstration toolkit for [Fine Structures](https://github.com/fine-strucutures/prime-materials).  Although [lib2x3](http://https://github.com/fine-structures/fine.SDK/tree/main/lib2x3) is a pure Go library, [`gpython`](http://github.com/go-python/gpython) is used to embed and expose it.  This means scripting is easy  — see for yourself in the [first tutorial](https://github.com/fine-structures/fine.SDK/blob/main/cmd/go2x3/learn/01-foundations.py) as you follow along in its [output](https://github.com/fine-structures/fine.SDK/blob/main/cmd/go2x3/learn/gold/01-foundations.txt).


## Releases

| Version   | Description                                                                               |
|:---------:|:-------------------------------------------------------------------------------------------------|
| v1.2022.1 | - traces-based particle catalog index  <br/> - conventional (non-canonic) vertex-based graph encoding  <br/> - introducing early tricodes   |
| v1.2022.2 | - refactor graph canonicalization  <br/> - refactor tricode console output |
| v1.2022.3 | - graph ascii art <br/> - refactor tricode encoding |
| v1.2023.1 | - refactor and cleanup |
| v1.2023.2 | - edge traces factorization (WIP) |
| v1.2023.3 | - edge traces factorization: all traces normalized |
| v1.2023.4 | - switched to vertex group factorization  |
| v1.2024.1 | - rename and copy edits  |


## Hot Topics

- Explore Traces normalization (to 0..1 non-linear transformations) 
    - Normalize each term by 1/Ci^2 or 3 and then sum (area and volume packing!)
    - Maybe the odd TracesTerms pack into open loops while even TracesTerms pack into closed loops?
    - [Hopf mapping](https://www.youtube.com/watch?v=PYR9worLEGo)
    - 3D volume packing of [Trapezo-Rhombic Dodecahedra](https://mathworld.wolfram.com/Trapezo-RhombicDodecahedron.html)
        - [Wikipedia](https://en.wikipedia.org/wiki/Trapezo-rhombic_dodecahedron)
        - [Cosmic evidence](https://www.cosmic-core.org/free/article-261-astronomy-the-geometry-of-galactic-clusters-part-2/)  
    - [Rydberg constant](https://en.wikipedia.org/wiki/Rydberg_constant)
    
- p-adic numbers & visualization: 
    - https://www.youtube.com/watch?v=tRaq4aYPzCc
    - https://en.wikipedia.org/wiki/P-adic_number
    - https://im.icerm.brown.edu/portfolio/apollonian-
    
- Consider y8d:
    - the odd cycles can only come from passing through one the two loops in this graph.

- Factor Traces into "prime basis vector" (count of each prime dot prime[i])

- There seems to be a clear path to "2-bit encodings" -- see GraphOp
   1) Graph builder walks thru all constructions.
   2) Because the walker is a canoinic walk, when an unwitnessed graph appears, assign it a new Traces ID (uint64).
   3) When a vertex length completes, for each Traces ID, choose the graph that most suitable canonic graph (most positive edge count?)
   4) The command list that builds the canonic graph can now be reduced to 2-bits per GraphOp (removing the VtxSlot operand) -- this now becomes the canonic graph encoding -- or the graph Traces ID itself.
   5) Browsing a TracesID could be visualls seeing all its variants
   
   // A graph encoding 
   
   // Allows the canonic traces ID to be found for any graph (by using the db)
   Traces => CanonicTracesID
   ...
   
   // Allows a canonic encoding for any encoding (without any computation) 
   CanonicEncoding => CanonicTracesID
   ...
   
   CanonicTracesID (uint64) => Traces, CanonicEncoding
        GraphEnc1
        GraphEnc2
        GraphEnc3
        ...
    ...
        
        
   
    if X is complete: 
       for each edge socket:
            for each GraphOp type:
                apply the op to the socket
        
   2)    For each GraphOp type, trying each graph op if possible.
   
   Plan:
   1) emit all "pure" (positive) graphs in canonic order (see OpCode)
   