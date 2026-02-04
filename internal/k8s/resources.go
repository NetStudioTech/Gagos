package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// ResourceDetail contains the YAML representation of a resource
type ResourceDetail struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	YAML      string `json:"yaml"`
}

// GetPod returns a single pod's details as YAML
func GetPod(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Clean up managed fields for cleaner YAML
	pod.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(pod)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Pod",
		Name:      pod.Name,
		Namespace: pod.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

// PatchPod updates a pod with the provided YAML
func PatchPod(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	// Convert YAML to JSON for strategic merge patch
	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.CoreV1().Pods(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

// DeletePod deletes a pod
func DeletePod(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	return clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetService returns a single service's details as YAML
func GetService(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	svc.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(svc)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Service",
		Name:      svc.Name,
		Namespace: svc.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

// PatchService updates a service with the provided YAML
func PatchService(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.CoreV1().Services(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

// DeleteService deletes a service
func DeleteService(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	return clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetDeployment returns a single deployment's details as YAML
func GetDeployment(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	dep, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	dep.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(dep)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Deployment",
		Name:      dep.Name,
		Namespace: dep.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

// PatchDeployment updates a deployment with the provided YAML
func PatchDeployment(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

// DeleteDeployment deletes a deployment
func DeleteDeployment(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	return clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetConfigMap returns a single configmap's details as YAML
func GetConfigMap(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cm.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(cm)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "ConfigMap",
		Name:      cm.Name,
		Namespace: cm.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

// PatchConfigMap updates a configmap with the provided YAML
func PatchConfigMap(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

// DeleteConfigMap deletes a configmap
func DeleteConfigMap(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	return clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetSecret returns a single secret's details as YAML (values base64 encoded)
func GetSecret(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	secret.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(secret)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Secret",
		Name:      secret.Name,
		Namespace: secret.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

// PatchSecret updates a secret with the provided YAML
func PatchSecret(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.CoreV1().Secrets(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

// DeleteSecret deletes a secret
func DeleteSecret(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	return clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetNode returns a single node's details as YAML
func GetNode(ctx context.Context, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	node, err := clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	node.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(node)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind: "Node",
		Name: node.Name,
		YAML: string(yamlBytes),
	}, nil
}

// GetNamespace returns a single namespace's details as YAML
func GetNamespace(ctx context.Context, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	ns, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ns.ManagedFields = nil

	yamlBytes, err := yaml.Marshal(ns)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind: "Namespace",
		Name: ns.Name,
		YAML: string(yamlBytes),
	}, nil
}

// DeleteNamespace deletes a namespace
func DeleteNamespace(ctx context.Context, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	return clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

// GetPodLogs returns logs from a pod
func GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	if clientset == nil {
		return "", fmt.Errorf("kubernetes client not initialized")
	}

	opts := &corev1.PodLogOptions{}
	if container != "" {
		opts.Container = container
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	result, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// ScaleDeployment scales a deployment to the specified replicas
func ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

// RestartDeployment triggers a rolling restart by updating an annotation
func RestartDeployment(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/restartedAt": metav1.Now().Format("2006-01-02T15:04:05Z"),
					},
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

// ========== ServiceAccount ==========

func GetServiceAccount(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	sa.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(sa)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "ServiceAccount",
		Name:      sa.Name,
		Namespace: sa.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func DeleteServiceAccount(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== PersistentVolume ==========

func GetPersistentVolume(ctx context.Context, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	pv, err := clientset.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	pv.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(pv)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind: "PersistentVolume",
		Name: pv.Name,
		YAML: string(yamlBytes),
	}, nil
}

func DeletePersistentVolume(ctx context.Context, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== PersistentVolumeClaim ==========

func GetPersistentVolumeClaim(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	pvc.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(pvc)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "PersistentVolumeClaim",
		Name:      pvc.Name,
		Namespace: pvc.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func PatchPersistentVolumeClaim(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

func DeletePersistentVolumeClaim(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== Ingress ==========

func GetIngress(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	ing, err := clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ing.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(ing)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Ingress",
		Name:      ing.Name,
		Namespace: ing.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func PatchIngress(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.NetworkingV1().Ingresses(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

func DeleteIngress(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== DaemonSet ==========

func GetDaemonSet(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ds.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(ds)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "DaemonSet",
		Name:      ds.Name,
		Namespace: ds.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func PatchDaemonSet(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

func DeleteDaemonSet(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func RestartDaemonSet(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/restartedAt": metav1.Now().Format("2006-01-02T15:04:05Z"),
					},
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

// ========== StatefulSet ==========

func GetStatefulSet(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	ss, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ss.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(ss)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "StatefulSet",
		Name:      ss.Name,
		Namespace: ss.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func PatchStatefulSet(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

func DeleteStatefulSet(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

func RestartStatefulSet(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/restartedAt": metav1.Now().Format("2006-01-02T15:04:05Z"),
					},
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}

// ========== Job ==========

func GetJob(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	job.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(job)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Job",
		Name:      job.Name,
		Namespace: job.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func DeleteJob(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	propagation := metav1.DeletePropagationBackground
	return clientset.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
}

// ========== CronJob ==========

func GetCronJob(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	cj, err := clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cj.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(cj)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "CronJob",
		Name:      cj.Name,
		Namespace: cj.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func PatchCronJob(ctx context.Context, namespace, name string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	jsonBytes, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err = clientset.BatchV1().CronJobs(namespace).Patch(ctx, name, types.StrategicMergePatchType, jsonBytes, metav1.PatchOptions{})
	return err
}

func DeleteCronJob(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== ReplicaSet ==========

func GetReplicaSet(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	rs, err := clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	rs.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(rs)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "ReplicaSet",
		Name:      rs.Name,
		Namespace: rs.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

func DeleteReplicaSet(ctx context.Context, namespace, name string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}
	return clientset.AppsV1().ReplicaSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// ========== Event ==========

func GetEvent(ctx context.Context, namespace, name string) (*ResourceDetail, error) {
	if clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}

	event, err := clientset.CoreV1().Events(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	event.ManagedFields = nil
	yamlBytes, err := yaml.Marshal(event)
	if err != nil {
		return nil, err
	}

	return &ResourceDetail{
		Kind:      "Event",
		Name:      event.Name,
		Namespace: event.Namespace,
		YAML:      string(yamlBytes),
	}, nil
}

// ========== Create Functions ==========

// CreateDeployment creates a new Deployment from YAML
func CreateDeployment(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var deployment appsv1.Deployment
	if err := yaml.Unmarshal([]byte(yamlContent), &deployment); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.AppsV1().Deployments(namespace).Create(ctx, &deployment, metav1.CreateOptions{})
	return err
}

// CreateService creates a new Service from YAML
func CreateService(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var svc corev1.Service
	if err := yaml.Unmarshal([]byte(yamlContent), &svc); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.CoreV1().Services(namespace).Create(ctx, &svc, metav1.CreateOptions{})
	return err
}

// CreateConfigMap creates a new ConfigMap from YAML
func CreateConfigMap(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var cm corev1.ConfigMap
	if err := yaml.Unmarshal([]byte(yamlContent), &cm); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metav1.CreateOptions{})
	return err
}

// CreateSecret creates a new Secret from YAML
func CreateSecret(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var secret corev1.Secret
	if err := yaml.Unmarshal([]byte(yamlContent), &secret); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{})
	return err
}

// CreateIngress creates a new Ingress from YAML
func CreateIngress(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var ing networkingv1.Ingress
	if err := yaml.Unmarshal([]byte(yamlContent), &ing); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.NetworkingV1().Ingresses(namespace).Create(ctx, &ing, metav1.CreateOptions{})
	return err
}

// CreatePod creates a new Pod from YAML
func CreatePod(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var pod corev1.Pod
	if err := yaml.Unmarshal([]byte(yamlContent), &pod); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
	return err
}

// CreateCronJob creates a new CronJob from YAML
func CreateCronJob(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var cj batchv1.CronJob
	if err := yaml.Unmarshal([]byte(yamlContent), &cj); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.BatchV1().CronJobs(namespace).Create(ctx, &cj, metav1.CreateOptions{})
	return err
}

// CreateJob creates a new Job from YAML
func CreateJob(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var job batchv1.Job
	if err := yaml.Unmarshal([]byte(yamlContent), &job); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.BatchV1().Jobs(namespace).Create(ctx, &job, metav1.CreateOptions{})
	return err
}

// CreatePersistentVolumeClaim creates a new PVC from YAML
func CreatePersistentVolumeClaim(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var pvc corev1.PersistentVolumeClaim
	if err := yaml.Unmarshal([]byte(yamlContent), &pvc); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &pvc, metav1.CreateOptions{})
	return err
}

// CreateServiceAccount creates a new ServiceAccount from YAML
func CreateServiceAccount(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var sa corev1.ServiceAccount
	if err := yaml.Unmarshal([]byte(yamlContent), &sa); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &sa, metav1.CreateOptions{})
	return err
}

// CreateDaemonSet creates a new DaemonSet from YAML
func CreateDaemonSet(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var ds appsv1.DaemonSet
	if err := yaml.Unmarshal([]byte(yamlContent), &ds); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.AppsV1().DaemonSets(namespace).Create(ctx, &ds, metav1.CreateOptions{})
	return err
}

// CreateStatefulSet creates a new StatefulSet from YAML
func CreateStatefulSet(ctx context.Context, namespace string, yamlContent string) error {
	if clientset == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	var ss appsv1.StatefulSet
	if err := yaml.Unmarshal([]byte(yamlContent), &ss); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	_, err := clientset.AppsV1().StatefulSets(namespace).Create(ctx, &ss, metav1.CreateOptions{})
	return err
}
