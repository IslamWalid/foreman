package depgraph

const (
    notVisited vertixStatus = 0
    currentlyVisiting vertixStatus = 1
    visited vertixStatus = 2
)

type vertixStatus int
type DepGraph map[string][]string

// Check if graph is cyclic.
func (g DepGraph) IsCyclic() bool {
    cyclic := false
    state := make(map[string]vertixStatus)

    var dfs func(string)
    dfs = func(vertix string) {
        if state[vertix] == visited {
            return
        }

        if state[vertix] == currentlyVisiting {
            cyclic = true
            return
        }

        state[vertix] = currentlyVisiting
        for _, child := range g[vertix] {
            dfs(child)
        }
        state[vertix] = visited
    }

    for vertix := range g {
        dfs(vertix)
    }

    return cyclic
}

// Topologically sort the dependency graph.
func (g DepGraph) TopSort() []string {
    out := make([]string, 0)
    state := make(map[string]vertixStatus)

    var dfs func(string)
    dfs = func(vertix string) {
        if state[vertix] == visited {
            return
        }

        state[vertix] = visited
        for _, child := range g[vertix] {
            dfs(child)
        }
        out = append(out, vertix)
    }

    for vertix := range g {
        dfs(vertix)
    }

    return out
}
