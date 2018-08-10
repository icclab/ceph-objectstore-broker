package broker

import (
	"errors"
	"github.com/pivotal-cf/brokerapi"
	"strconv"
	"strings"
)

func (b *Broker) instanceExists(instID string) bool {
	_, err := b.S3.GetObjectInfo(BucketName, getInstanceObjName(instID))
	return err == nil
}

func (b *Broker) bindingExists(instID string, bindID string) bool {
	_, err := b.S3.GetObjectInfo(BucketName, getBindObjName(instID, bindID))
	return err == nil
}

//Returns the number of provisioned instances
func (b *Broker) provisionCount() int {
	objs, done := b.S3.GetObjects(BucketName, instancePrefix, false)
	defer close(done)
	count := 0
	for range objs {
		count++
	}
	return count
}

//Returns true if the provisioned instance has any binds
func (b *Broker) hasBinds(instID string) bool {
	objs, done := b.S3.GetObjects(BucketName, getInstanceObjName(instID)+"/", false)
	defer close(done)
	for range objs {
		return true
	}
	return false
}

func (b *Broker) getPlan(planID string) (*brokerapi.ServicePlan, error) {
	for _, p := range b.ServiceConfig[0].Plans {
		if p.ID == planID {
			return &p, nil
		}
	}

	return nil, errors.New("Plan with ID '" + planID + "' not found")
}

func (b *Broker) getPlanQuota(planID string) (int, error) {
	p, err := b.getPlan(planID)
	if err != nil {
		return -1, err
	}

	i, err := strconv.Atoi(p.Metadata.AdditionalMetadata["quotaMB"].(string))
	if err != nil {
		return -1, err
	}

	return i, nil
}

func createTenantID(instanceID string) string {
	return strings.Replace(instanceID, "-", "", -1)
}

//Converts the instance ID into the object name format
func getInstanceObjName(instID string) string {
	return instancePrefix + instID
}

//Converts the instance and binding IDs into the object name format
func getBindObjName(instID string, bindID string) string {
	return instancePrefix + instID + "/" + bindID
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
