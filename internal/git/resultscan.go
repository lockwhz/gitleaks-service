// Helpers for working with scan results at application layer.
package git

// FlattenFindings percorre o slice de BranchScanResult e devolve todos os
// findings em uma lista única.  
// Útil para relatórios ou testes em que você não precisa manter a divisão
// por branch, mas quer iterar em todas as ocorrências.
func FlattenFindings(results []BranchScanResult) []Finding {
    size := 0
    for _, br := range results {
        size += len(br.Findings)
    }
    out := make([]Finding, 0, size)
    for _, br := range results {
        out = append(out, br.Findings...)
    }
    return out
}

// CountByRule devolve um mapa ruleID → quantidade, somando todos os branches.
func CountByRule(results []BranchScanResult) map[string]int {
    tally := make(map[string]int)
    for _, br := range results {
        for _, f := range br.Findings {
            tally[f.RuleID]++
        }
    }
    return tally
}
