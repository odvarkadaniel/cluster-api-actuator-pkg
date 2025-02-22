package framework

import (
	"context"
	"fmt"

	kappsapi "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDaemonset gets deployment object by name and namespace.
func GetDaemonset(c client.Client, name, namespace string) (*kappsapi.DaemonSet, error) {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	d := &kappsapi.DaemonSet{}

	if err := wait.PollImmediate(RetryMedium, WaitShort, func() (bool, error) {
		if err := c.Get(context.TODO(), key, d); err != nil {
			klog.Errorf("Error querying api for DaemonSet object %q: %v, retrying...", name, err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting DaemonSet %q: %w", name, err)
	}

	return d, nil
}

// DeleteDaemonset deletes the specified deployment.
func DeleteDaemonset(c client.Client, deployment *kappsapi.DaemonSet) error {
	return wait.PollImmediate(RetryMedium, WaitShort, func() (bool, error) {
		if err := c.Delete(context.TODO(), deployment); err != nil {
			klog.Errorf("error querying api for DaemonSet object %q: %v, retrying...", deployment.Name, err)
			return false, nil
		}

		return true, nil
	})
}

// UpdateDaemonset updates the specified deployment.
func UpdateDaemonset(c client.Client, name, namespace string, updated *kappsapi.DaemonSet) error {
	return wait.PollImmediate(RetryMedium, WaitMedium, func() (bool, error) {
		d, err := GetDeployment(c, name, namespace)
		if err != nil {
			klog.Errorf("Error getting DaemonSet: %v", err)
			return false, nil
		}
		if err := c.Patch(context.TODO(), d, client.MergeFrom(updated)); err != nil {
			klog.Errorf("error patching DaemonSet object %q: %v, retrying...", name, err)
			return false, nil
		}

		return true, nil
	})
}

// IsDaemonsetAvailable returns true if the deployment has one or more available replicas.
func IsDaemonsetAvailable(c client.Client, name, namespace string) bool {
	if err := wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		d, err := GetDaemonset(c, name, namespace)
		if err != nil {
			klog.Errorf("Error getting DaemonSet: %v", err)
			return false, nil
		}
		if d.Status.NumberAvailable == 0 {
			klog.Errorf("DaemonSet %q is not available. Status: %s",
				d.Name, daemonsetInfo(d))

			return false, nil
		}
		klog.Infof("DaemonSet %q is available. Status: %s",
			d.Name, daemonsetInfo(d))

		return true, nil
	}); err != nil {
		klog.Errorf("Error checking IsDaemonsetAvailable: %v", err)
		return false
	}

	return true
}

func daemonsetInfo(d *kappsapi.DaemonSet) string {
	return fmt.Sprintf("(ready: %d, available: %d, unavailable: %d)",
		d.Status.NumberReady, d.Status.NumberAvailable, d.Status.NumberUnavailable)
}
