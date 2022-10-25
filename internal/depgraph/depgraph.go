package depgraph

const (
    // notVisited marks the vertices that is not visited yet.
    notVisited vertixStatus = 0

    // currentlyVisiting marks the vertices that is currently being visited.
    currentlyVisiting vertixStatus = 1

    // visited marks the vertices that has been already visited.
    visited vertixStatus = 2
)

// vertixStatus represents the current status of a vertix.
type vertixStatus int

// DepGraph represents the graph of the services and their dependenis.
type DepGraph map[string][]string

// IsCyclic check if the graph has cycles.
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

// TopSort topologically sort the dependency graph.
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
