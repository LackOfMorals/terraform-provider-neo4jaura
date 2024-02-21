package client

// todo source instance id && source snapshot id
type PostInstanceRequest struct {
	Version          string  `json:"version"`
	Region           string  `json:"region"`
	Memory           string  `json:"memory"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	TenantId         string  `json:"tenant_id"`
	CloudProvider    string  `json:"cloud_provider"`
	SourceInstanceId *string `json:"source_instance_id,omitempty"`
	SourceSnapshotId *string `json:"source_snapshot_id,omitempty"`
}

type PatchInstanceRequest struct {
	Name   *string `json:"name"`
	Memory *string `json:"memory"`
}
