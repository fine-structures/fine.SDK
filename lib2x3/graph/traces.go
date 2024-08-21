package graph


var Primes = []int64{
    1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 
    101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199, 
    211, 223, 227, 229, 233, 239, 241, 251, 257, 263, 269, 271, 277, 281, 283, 293,
}

var PrimesByID = map[int64]int64{}

func init() {
    for i, p := range Primes {
        PrimesByID[int64(i)] = p
    }
    
}
// cycleNum: 1, 2, ...
// 
// func (term *TracesTerm) AccumulateCycle(cycleLen int, cycleCount int64) {

//         // for each factor of the cycle count, increment the corresponding base prime count
//     maxFactor := int64(cycleCount / 2)
//     for fi := int64(2); fi <= maxFactor; i++ {
//         result := cycleCount / fi
//         if result * fi == cycleCount {
//             cycleCount = result
//             idx := slices.BinarySearch(term.TX_PrimeBases, func(idx int) {
//             })
            
//         }
//     }
        
// }

// 
func Factorize(factorCounts *[]int64, primeFactors *[]int64, x int64) {
    if x < 0 {
        x = -x
    }
    hasFactors := false
    
     // Pull out primes until we can't
    for Pi := 1; x > 1; Pi++ { 
        primeFactor := Primes[Pi]
        
        // factor out as many of this prime factor as possible
        factorCount := int64(0)
        for {
            result := x / primeFactor
            if result * primeFactor != x { // easier to multiply than divide; symmetry identified
                break
            }
            hasFactors = true
            factorCount++
            x = result
        }
        
        // any modulus digit that is 
        if factorCount > 0 {
            *primeFactors = append(*primeFactors, primeFactor)
            *factorCounts = append(*factorCounts, factorCount)
        }
        
        // stop when this (and larger) factors couldn't  be a factor
        if 2 * primeFactor > x {
            break
        }
    
    }
    
    // if we have a prime factor left, add it
    if !hasFactors || x > 1 {
        *primeFactors = append(*primeFactors, x)
        *factorCounts = append(*factorCounts, 1)
    }

}