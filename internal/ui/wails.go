// File: internal/ui/wails.go
package ui

import (
    "fmt"
    "guitarHetic/internal/config"
    "sort"
)

type UniRange struct {
    Universe int      `json:"universe"`
    Ranges   []string `json:"ranges"`
}

type Handlers struct {
    Controllers []string
    CtrlMap     map[string]map[int][][2]int
}

func BuildModel(cfg *config.Config) ([]string, map[string]map[int][][2]int) {
    ctrls := make(map[string]map[int][][2]int)
    for _, e := range cfg.RoutingTable {
        if ctrls[e.IP] == nil {
            ctrls[e.IP] = make(map[int][][2]int)
        }
        ctrls[e.IP][e.Universe] = append(ctrls[e.IP][e.Universe], [2]int{e.EntityID, e.EntityID})
    }
    for _, uniMap := range ctrls {
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
    ips := make([]string, 0, len(ctrls))
    for ip := range ctrls {
        ips = append(ips, ip)
    }
    sort.Strings(ips)
    return ips, ctrls
}

func (h *Handlers) GetControllers() []string {
    return h.Controllers
}

func (h *Handlers) GetDetails(ip string) []UniRange {
    entries := h.CtrlMap[ip]
    out := make([]UniRange, 0, len(entries))
    for u, rs := range entries {
        parts := make([]string, len(rs))
        for i, r := range rs {
            parts[i] = fmt.Sprintf("%d–%d", r[0], r[1])
        }
        out = append(out, UniRange{Universe: u, Ranges: parts})
    }
    sort.Slice(out, func(i, j int) bool { return out[i].Universe < out[j].Universe })
    return out
}

func (h *Handlers) Reload() error {
    cfg, err := config.Load("internal/config/routing.csv")
    if err != nil {
        return err
    }
    h.Controllers, h.CtrlMap = BuildModel(cfg)
    return nil
}
