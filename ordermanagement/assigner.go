package ordermanagement

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func CallAssigner(path string, in AssignerInput) (AssignerOutput, error) {
	req, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	req = append(req, '\n') // readln

	cmd := exec.Command(path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	_, err = stdin.Write(req)
	_ = stdin.Close()
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("assigner failed: %v, stderr: %s", err, stderr.String())
	}

	var out AssignerOutput
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &out); err != nil {
		return nil, fmt.Errorf("bad JSON from assigner: %v\nraw: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return out, nil
}
/* 
func main() {
	in := AssignerInput{
		HallRequests: [][]bool{
			{false, false},
			{true, false},
			{false, true},
			{false, false},
		},
		States: map[string]ElevatorState{
			"id_1": {
				Behaviour:   BehIdle,
				Floor:       0,
				Direction:   DirStop,
				CabRequests: []bool{false, false, false, false},
			},"id_2": {
				Behaviour:   BehMoving,
				Floor:       2,
				Direction:   DirUp,
				CabRequests: []bool{false, false, false, false},
			},
			
		},
	}

	out, err := CallAssigner("../hall_request_assigner/hall_request_assigner", in)
	if err != nil {
		panic(err)
	}

	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
} */