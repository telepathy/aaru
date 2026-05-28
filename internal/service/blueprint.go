package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"aaru/internal/model"
	"aaru/internal/store"
)

type BlueprintService struct{ store *store.DBStore }

func NewBlueprintService(s *store.DBStore) *BlueprintService { return &BlueprintService{store: s} }

func genToken() string { b := make([]byte, 16); rand.Read(b); return hex.EncodeToString(b) }

type BlueprintInput struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Nodes       []model.BlueprintNode `json:"nodes"`
	Edges       []model.BlueprintEdge `json:"edges"`
}

func (b *BlueprintService) Create(in *BlueprintInput) (*model.PromotionBlueprint, error) {
	if err := b.prepareAndValidate(in); err != nil {
		return nil, err
	}
	bp := &model.PromotionBlueprint{Name: in.Name, Description: in.Description}
	if err := b.store.CreateBlueprint(bp); err != nil {
		return nil, err
	}
	for i := range in.Nodes { in.Nodes[i].BlueprintID = bp.ID }
	for i := range in.Edges { in.Edges[i].BlueprintID = bp.ID }
	if err := b.store.CreateNodes(in.Nodes); err != nil {
		return nil, fmt.Errorf("create nodes: %w", err)
	}
	if err := b.store.CreateEdges(in.Edges); err != nil {
		return nil, fmt.Errorf("create edges: %w", err)
	}
	bp2, _ := b.store.GetBlueprint(bp.ID)
	return bp2, nil
}

func (b *BlueprintService) Update(id uint, in *BlueprintInput) (*model.PromotionBlueprint, error) {
	if err := b.prepareAndValidate(in); err != nil {
		return nil, err
	}
	bp, err := b.store.GetBlueprint(id)
	if err != nil { return nil, err }
	bp.Name = in.Name; bp.Description = in.Description
	b.store.UpdateBlueprint(bp)
	b.store.DeleteNodesByBlueprint(id)
	b.store.DeleteEdgesByBlueprint(id)
	for i := range in.Nodes { in.Nodes[i].BlueprintID = id }
	for i := range in.Edges { in.Edges[i].BlueprintID = id }
	b.store.CreateNodes(in.Nodes)
	b.store.CreateEdges(in.Edges)
	bp2, _ := b.store.GetBlueprint(id)
	return bp2, nil
}

// ensureApprovalRole 为指定环境创建/查找审批角色
func (b *BlueprintService) ensureApprovalRole(envCode, envName string) (*model.Role, error) {
	roleName := "approver-" + envCode
	roles, _ := b.store.ListRoles()
	for _, r := range roles {
		if r.Name == roleName {
			return &r, nil
		}
	}
	role := &model.Role{Name: roleName, Description: envName + " 环境审批角色（自动创建）"}
	if err := b.store.CreateRole(role); err != nil {
		return nil, err
	}
	// 自动授予 approve 权限
	b.store.SetRolePermissions(role.ID, []model.Permission{
		{DeployUnitCode: "*", Action: "approve"},
		{DeployUnitCode: "*", Action: "view"},
	})
	return role, nil
}

func (b *BlueprintService) prepareAndValidate(in *BlueprintInput) error {
	oldToNew := make(map[uint]uint)
	var next uint = 1
	for i := range in.Nodes {
		old := in.Nodes[i].ID
		in.Nodes[i].ID = next
		oldToNew[old] = next
		next++

		if in.Nodes[i].GateType == "api_hook" && in.Nodes[i].WebhookToken == "" {
			in.Nodes[i].WebhookToken = genToken()
		}
		if in.Nodes[i].GateType == "manual" {
			role, err := b.ensureApprovalRole(in.Nodes[i].EnvCode, in.Nodes[i].EnvName)
			if err != nil {
				return fmt.Errorf("创建审批角色失败: %w", err)
			}
			in.Nodes[i].ApproveRoleID = &role.ID
		}
	}
	for i := range in.Edges {
		in.Edges[i].FromNodeID = oldToNew[in.Edges[i].FromNodeID]
		in.Edges[i].ToNodeID = oldToNew[in.Edges[i].ToNodeID]
	}
	seenEnv := make(map[string]bool)
	for _, n := range in.Nodes {
		if n.EnvCode == "" {
			return fmt.Errorf("节点 %s 未选择环境", n.EnvName)
		}
		if seenEnv[n.EnvCode] {
			return fmt.Errorf("环境 %s 在蓝图中重复出现", n.EnvCode)
		}
		seenEnv[n.EnvCode] = true
	}
	seenEdge := make(map[string]bool)
	for _, e := range in.Edges {
		key := fmt.Sprintf("%d->%d", e.FromNodeID, e.ToNodeID)
		if seenEdge[key] {
			return fmt.Errorf("边 %d→%d 重复", e.FromNodeID, e.ToNodeID)
		}
		if e.FromNodeID == e.ToNodeID {
			return fmt.Errorf("不允许自环边（%d→%d）", e.FromNodeID, e.ToNodeID)
		}
		seenEdge[key] = true
	}
	if err := validateDAG(in.Nodes, in.Edges); err != nil {
		return err
	}
	return nil
}

func (b *BlueprintService) Get(id uint) (*model.PromotionBlueprint, error) { return b.store.GetBlueprint(id) }

func (b *BlueprintService) List() ([]map[string]interface{}, error) {
	bps, _ := b.store.ListBlueprints()
	var r []map[string]interface{}
	for _, bp := range bps {
		nodes, _ := b.store.GetBlueprintNodes(bp.ID)
		edges, _ := b.store.GetBlueprintEdges(bp.ID)
		r = append(r, map[string]interface{}{
			"id": bp.ID, "name": bp.Name, "description": bp.Description,
			"node_count": len(nodes), "edge_count": len(edges),
			"created_at": bp.CreatedAt, "updated_at": bp.UpdatedAt,
		})
	}
	return r, nil
}
func (b *BlueprintService) Delete(id uint) error { return b.store.DeleteBlueprint(id) }

func (b *BlueprintService) GetSourceNodeIDs(bpID uint) ([]uint, error) {
	nodes, _ := b.store.GetBlueprintNodes(bpID)
	edges, _ := b.store.GetBlueprintEdges(bpID)
	hasIn := make(map[uint]bool)
	for _, e := range edges { hasIn[e.ToNodeID] = true }
	var src []uint
	for _, n := range nodes { if !hasIn[n.ID] { src = append(src, n.ID) } }
	return src, nil
}
func (b *BlueprintService) GetParentNodeIDs(bpID, nodeID uint) ([]uint, error) {
	edges, _ := b.store.GetBlueprintEdges(bpID)
	var p []uint
	for _, e := range edges { if e.ToNodeID == nodeID { p = append(p, e.FromNodeID) } }
	return p, nil
}
func (b *BlueprintService) GetChildNodeIDs(bpID, nodeID uint) ([]uint, error) {
	edges, _ := b.store.GetBlueprintEdges(bpID)
	var c []uint
	for _, e := range edges { if e.FromNodeID == nodeID { c = append(c, e.ToNodeID) } }
	return c, nil
}
func (b *BlueprintService) IsSinkNode(bpID, nodeID uint) (bool, error) {
	edges, _ := b.store.GetBlueprintEdges(bpID)
	for _, e := range edges { if e.FromNodeID == nodeID { return false, nil } }
	return true, nil
}

func validateDAG(nodes []model.BlueprintNode, edges []model.BlueprintEdge) error {
	nodeIDs := make(map[uint]bool)
	for _, n := range nodes { nodeIDs[n.ID] = true }
	for _, e := range edges {
		if !nodeIDs[e.FromNodeID] { return fmt.Errorf("边从节点%d出发，但该节点不存在", e.FromNodeID) }
		if !nodeIDs[e.ToNodeID] { return fmt.Errorf("边指向节点%d，但该节点不存在", e.ToNodeID) }
	}
	inDegree := make(map[uint]int); adj := make(map[uint][]uint)
	for _, n := range nodes { inDegree[n.ID] = 0 }
	for _, e := range edges {
		adj[e.FromNodeID] = append(adj[e.FromNodeID], e.ToNodeID)
		inDegree[e.ToNodeID]++
	}
	var q []uint
	for _, n := range nodes { if inDegree[n.ID] == 0 { q = append(q, n.ID) } }
	visited := 0
	for len(q) > 0 {
		u := q[0]; q = q[1:]; visited++
		for _, v := range adj[u] {
			inDegree[v]--
			if inDegree[v] == 0 { q = append(q, v) }
		}
	}
	if visited != len(nodes) {
		return fmt.Errorf("蓝图中存在循环依赖（仅%d/%d个节点可达）", visited, len(nodes))
	}
	return nil
}
