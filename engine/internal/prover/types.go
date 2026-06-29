package prover

type Proof struct {
	ID       string   `json:"id"`
	Agent    string   `json:"agent"`
	PH       string   `json:"proof_hash"`
	IH       string   `json:"input_hash"`
	OH       string   `json:"output_hash"`
	MH       string   `json:"model_hash"`
	Root     string   `json:"merkle_root"`
	Path     []string `json:"merkle_path"`
	Idx      int      `json:"leaf_index"`
	TS       int64    `json:"timestamp"`
	Valid    bool     `json:"valid"`
	Revoked  bool     `json:"revoked"`
	UseCase  string   `json:"use_case"`
}

type MerkleNode struct {
	Hash  [32]byte
	Left  *MerkleNode
	Right *MerkleNode
}

type ProofBundle struct {
	Root  string   `json:"root"`
	Leafs []string `json:"leafs"`
	Count int      `json:"count"`
}
