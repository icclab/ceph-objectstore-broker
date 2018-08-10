package s3

import (
	"github.com/minio/minio-go"
	"io"
	"strings"
)

type S3 struct {
	conn *minio.Client
}

//Setups a connection to S3. Must be called before any other function
func (s3 *S3) Connect(endpoint string, accessKey string, secretKey string, secure bool) error {
	if secure {
		endpoint = strings.Replace(endpoint, "https://", "", 1)
	} else {
		endpoint = strings.Replace(endpoint, "http://", "", 1)
	}

	c, err := minio.New(endpoint, accessKey, secretKey, secure)
	s3.conn = c
	return err
}

func (s3 *S3) CreateBucket(name string) error {
	return s3.conn.MakeBucket(name, "")
}

func (s3 *S3) BucketExists(bucketName string) (bool, error) {
	e, err := s3.conn.BucketExists(bucketName)
	return e, err
}

func (s3 *S3) DeleteBucket(name string) error {
	return s3.conn.RemoveBucket(name)
}

func (s3 *S3) PutObject(bucketName string, objName string, data string) error {
	r := strings.NewReader(data)
	_, err := s3.conn.PutObject(bucketName, objName, r, r.Size(), minio.PutObjectOptions{})
	return err
}

func (s3 *S3) PutObjectWithMetadata(bucketName string, objName string, data string, metadata map[string]string) error {
	r := strings.NewReader(data)
	_, err := s3.conn.PutObject(bucketName, objName, r, r.Size(), minio.PutObjectOptions{UserMetadata: metadata})
	return err
}

//Gets the specified object and returns the contained data as a string
func (s3 *S3) GetObjectString(bucketName string, objName string) (string, error) {
	o, err := s3.conn.GetObject(bucketName, objName, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}

	oi, err := o.Stat()
	if err != nil {
		return "", err
	}

	b := make([]byte, oi.Size)
	_, err = o.Read(b)
	if err != nil && err != io.EOF {
		return "", err
	}

	return string(b), nil
}

func (s3 *S3) GetObjectInfo(bucketName string, objName string) (*minio.ObjectInfo, error) {
	oi, err := s3.conn.StatObject(bucketName, objName, minio.StatObjectOptions{})
	return &oi, err
}

//GetObjects Returns a channel of object information and another channeling to control the underlying goroutine
//
//'recursive' is whether all 'directories' under the given prefix are given, or only the directory directly under it
//
//The second returned channel should be closed by the user when done using the object information channel
func (s3 *S3) GetObjects(bucketName string, objectPrefix string, recursive bool) (<-chan minio.ObjectInfo, chan struct{}) {
	doneCh := make(chan struct{})
	return s3.conn.ListObjectsV2(bucketName, objectPrefix, recursive, doneCh), doneCh
}

func (s3 *S3) DeleteObject(bucketName string, objName string) error {
	return s3.conn.RemoveObject(bucketName, objName)
}
