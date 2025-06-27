// File: internal/ui/model.go
package ui

import (
    "guitarHetic/internal/config"
    "sort"
)

type UniRange struct {
    Universe int
    Ranges   [][2]int
}

func BuildModel(cfg *config.Config) ([]string, map[string]map[int][][2]int) {
    controllers := make(map[string]map[int][][2]int)
    for _, e := range cfg.RoutingTable {
        ip := e.IP
        if controllers[ip] == nil {
            controllers[ip] = make(map[int][][2]int)
        }
        controllers[ip][e.Universe] = append(controllers[ip][e.Universe], [2]int{e.EntityID, e.EntityID})
    }
    for _, uniMap := range controllers {
        for u, ids := range uniMap {
            list := make([]int, len(ids))
            for i, p := range ids {
                list[i] = p[0]
            }
            sort.Ints(list)
            var ranges [][2]int
            start, end := list[0], list[0]
            for _, id := range list[1:] {
                if id == end+1 {
                    end = id
                } else {
                    ranges = append(ranges, [2]int{start, end})
                    start, end = id, id
                }
            }
            ranges = append(ranges, [2]int{start, end})
            uniMap[u] = ranges
        }
    }
    ips := make([]string, 0, len(controllers))
    for ip := range controllers {
        ips = append(ips, ip)
    }
    sort.Strings(ips)
    return ips, controllers
}
