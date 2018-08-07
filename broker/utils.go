package broker

import (
	"errors"
	"github.com/pivotal-cf/brokerapi"
	"strconv"
	"strings"
)

func createTenantID(instanceID string) string {
	return strings.Replace(instanceID, "-", "", -1)
}

func sliceContains(needle string, haystack []string) bool {
	for _, element := range haystack {
		if element == needle {
			return true
		}
	}
	return false
}

func getElementIndex(s string, slice []string) int {
	for i, x := range slice {
		if s == x {
			return i
		}
	}

	return -1
}

func removeFromSlice(s string, slice []string) []string {
	i := getElementIndex(s, slice)

	length := len(slice)
	t := slice[length-1]
	slice[length-1] = slice[i]
	slice[i] = t
	return slice[:length-1]
}

func getPlan(planID string, plans []brokerapi.ServicePlan) (*brokerapi.ServicePlan, error) {
	for _, p := range plans {
		if p.ID == planID {
			return &p, nil
		}
	}

	return nil, errors.New("Plan with ID '" + planID + "' not found")
}

func getPlanQuota(planID string, plans []brokerapi.ServicePlan) (int, error) {
	p, err := getPlan(planID, plans)
	if err != nil {
		return -1, err
	}

	i, err := strconv.Atoi(p.Metadata.AdditionalMetadata["quotaMB"].(string))
	if err != nil {
		return -1, err
	}

	return i, nil
}
