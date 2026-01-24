package hydrate

import "encoding/json"

type PlanReport struct {
	Ops []Op `json:"ops"`
}

func FormatOpsJSON(plan Plan) ([]byte, error) {
	payload := PlanReport{Ops: plan.Ops}
	return json.Marshal(payload)
}

func FormatOpsText(plan Plan) string {
	out := ""
	for i, op := range plan.Ops {
		line := string(op.Kind) + " " + op.RelPath
		if op.Kind == OpWriteFile && op.Content != nil {
			line += " (content)"
		}
		if op.Kind == OpSymlink {
			line += " -> " + op.Target
		}
		if i == 0 {
			out = line
		} else {
			out += "\n" + line
		}
	}
	return out
}
