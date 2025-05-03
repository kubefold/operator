package alphafold

type Input struct {
	Name       string     `json:"name"`
	Sequences  []Sequence `json:"sequences"`
	ModelSeeds []int      `json:"modelSeeds"`
	Dialect    string     `json:"dialect"`
	Version    int        `json:"version"`
}

type Sequence struct {
	Protein Protein `json:"protein"`
}

type Protein struct {
	ID          []string `json:"id"`
	Sequence    string   `json:"sequence"`
	UnpairedMSA string   `json:"unpairedMsa"`
	PairedMSA   string   `json:"pairedMsa"`
	Templates   []string `json:"templates"`
}
