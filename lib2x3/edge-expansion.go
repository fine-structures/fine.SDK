package lib2x3

// type EdgeTemplate byte
// const (
//     SingleEdge EdgeTemplate = iota
//     DownVertex
//     MobiusQuad
// )


type GraphState interface {
    PushState()
    PopState()
    
    GrowVtxOnEdge(va, vb VtxID)
    
    GrowSlotFace(va, vb VtxID)
    
    GrowMobiusEdge(va, vb VtxID)
    
    GrowVtxFromPole(v VtxID)
    
        
}

// Nice!  As we exapnd, if the current X state has already been traversed, that branch halts


func GrowActiveEdges() {


    // for {
        
    //     // First, consume open poles towards consuming open poles 

    //     // Va, Vb := activeEdgeSet.Dequeue()
    //     // activePoles.Add
        
    //     //AddEdge(Va, Vb, DownVertex, 
    // }

}

func GrowPoles() {

}