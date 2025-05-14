package models

type Verdict struct {
	Name string
	Code string
}

var (
	VerdictAC  = Verdict{Name: "Accepted", Code: "AC"}
	VerdictWA  = Verdict{Name: "Wrong Answer", Code: "WA"}
	VerdictCE  = Verdict{Name: "Compilation Error", Code: "CE"}
	VerdictRE  = Verdict{Name: "Runtime Error", Code: "RE"}
	VerdictTLE = Verdict{Name: "Time Limit Exceeded", Code: "TLE"}
	VerdictXX  = Verdict{Name: "Internal Error", Code: "XX"}
)
