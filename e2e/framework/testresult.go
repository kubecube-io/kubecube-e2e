package framework

import (
	"context"
	e2econstants "github.com/kubecube-io/kubecube-e2e/util/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
	"strings"
	"sync"
)

var TestResultInstance = TesultResult{}

type TesultResult struct {
	resultMap map[string]string
	lock      sync.RWMutex
}

func (t *TesultResult) Init() error {
	configMap, err := GetConfigMap(context.Background())
	if err != nil {
		return err
	}
	var testResultMap map[string]string
	data := configMap.Data
	if data != nil {
		err := yaml.Unmarshal([]byte(data[getTestResultKey()]), &testResultMap)
		if err != nil {
			return err
		}
	}
	if testResultMap == nil {
		testResultMap = make(map[string]string)
	}
	t.resultMap = testResultMap
	return nil
}

func (t *TesultResult) Get(key string) (string, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	val, ok := t.resultMap[key]
	return val, ok
}
func (t *TesultResult) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	delete(t.resultMap, key)
}
func (t *TesultResult) Set(key string, value string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.resultMap[key] = value
}

func (t *TesultResult) End() error {
	timeout, cancelFunc := context.WithTimeout(context.Background(), WaitTimeout)
	defer cancelFunc()
	return wait.PollUntilContextCancel(timeout, WaitInterval, false, func(ctx context.Context) (done bool, err error) {
		configMap, err := GetConfigMap(context.Background())
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		data, err := yaml.Marshal(t.resultMap)
		if err != nil {
			return false, err
		}
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data[getTestResultKey()] = string(data)
		err = PivotClusterClient.Direct().Update(context.Background(), configMap)
		return true, err
	})
}

func getTestResultKey() string {
	userName := strings.Join(TestUser, "-")
	return userName + "-" + e2econstants.ConfigMapTestResultKey
}
