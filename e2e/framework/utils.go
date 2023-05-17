package framework

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"
)

func PostFile(filename string, fieldName string, target_url string, header map[string]string) (*http.Response, error) {
	body_buf := bytes.NewBufferString("")
	body_writer := multipart.NewWriter(body_buf)

	// use the body_writer to write the Part headers to the buffer
	_, err := body_writer.CreateFormFile(fieldName, filename)
	if err != nil {
		clog.Warn("error writing to buffer, %v", err)
		return nil, err
	}

	// the file data will be the second part of the body
	fh, err := os.Open(filename)
	if err != nil {
		clog.Warn("error opening file, %v", err)
		return nil, err
	}
	// need to know the boundary to properly close the part myself.
	boundary := body_writer.Boundary()
	//close_string := fmt.Sprintf("\r\n--%s--\r\n", boundary)
	close_buf := bytes.NewBufferString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	// use multi-reader to defer the reading of the file data until
	// writing to the socket buffer.
	request_reader := io.MultiReader(body_buf, fh, close_buf)
	fi, err := fh.Stat()
	if err != nil {
		clog.Warn("Error Stating file: %s", filename)
		return nil, err
	}
	req, err := http.NewRequest("POST", target_url, request_reader)
	if err != nil {
		clog.Warn("post file error, %v", req)
		return nil, err
	}

	// Set headers for multipart, and Content Length
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)

	for k, v := range header {
		req.Header.Add(k, v)
	}

	req.ContentLength = fi.Size() + int64(body_buf.Len()) + int64(close_buf.Len())

	return http.DefaultClient.Do(req)
}

// CreateNamespace Create namespace
func CreateNamespace(baseName string, cli client.Client) (*v1.Namespace, error) {
	labels := map[string]string{
		"e2e-run":       string(uuid.NewUUID()),
		"e2e-framework": baseName,
	}
	name := fmt.Sprintf("kubecube-e2etest-%v-%v", baseName, strconv.Itoa(rand.Intn(10000)))
	namespaceObj := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	if err := wait.PollImmediate(WaitInterval, WaitTimeout, func() (bool, error) {
		err := cli.Direct().Create(context.TODO(), namespaceObj)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				// regenerate on conflict
				clog.Info("Namespace name %q was already taken, generate a new name and retry", namespaceObj.Name)
				namespaceObj.Name = fmt.Sprintf("kubecube-e2etest-%v-%v", name, strconv.Itoa(rand.Intn(10000)))
			} else {
				clog.Info("Unexpected error while creating namespace: %v", err)
			}
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, err
	}

	return namespaceObj, nil
}

// DeleteNamespace Delete Namespace
func DeleteNamespace(ns *v1.Namespace, cli client.Client) error {
	err := cli.Direct().Delete(context.TODO(), ns)
	if err != nil && !apierrors.IsNotFound(err) {
		clog.Error("error deleting namespace %s: %v", ns.Name, err)
		return err
	}
	if err = wait.Poll(WaitInterval, WaitTimeout,
		func() (bool, error) {
			var nsTemp v1.Namespace
			err := cli.Direct().Get(context.TODO(), types.NamespacedName{Name: ns.Name}, &nsTemp)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			}
			return false, nil
		}); err != nil {
		return err
	}
	return nil
}

func GetUser(user string) string {
	switch user {
	case UserAdmin:
		return Admin
	case UserProjectAdmin:
		return ProjectAdmin
	case UserTenantAdmin:
		return TenantAdmin
	case UserNormal:
		return User
	default:
		return Admin
	}
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}

	return false
}
