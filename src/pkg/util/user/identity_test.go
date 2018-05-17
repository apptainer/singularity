package user

import "testing"

func TestGetPwUID(t *testing.T) {
	_, err := GetPwUID(0)
	if err != nil {
		t.Errorf("Failed to retrieve information for UID 0")
	}
}

func TestGetPwNam(t *testing.T) {
	_, err := GetPwNam("root")
	if err != nil {
		t.Errorf("Failed to retrieve information for root user")
	}
}

func TestGetGrGID(t *testing.T) {
	_, err := GetGrGID(0)
	if err != nil {
		t.Errorf("Failed to retrieve information for GID 0")
	}
}

func TestGetGrNam(t *testing.T) {
	_, err := GetGrNam("root")
	if err != nil {
		t.Errorf("Failed to retrieve information for root group")
	}
}
