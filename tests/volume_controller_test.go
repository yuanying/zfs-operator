/*
Copyright 2020 O.Yuanying <yuanying@fraction.jp>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"context"
	"fmt"

	"github.com/mistifyio/go-zfs/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	zfsv1alpha1 "github.com/yuanying/zfs-operator/api/v1alpha1"
)

var _ = Describe("VolumeController", func() {

	Context("ZFS volume", func() {

		It("should create, update and delete ZVOL", func() {
			var err error
			zfsVolName := fmt.Sprintf("%s/%s", testZfsParent, "pvc-zfs1")
			ctx := context.Background()
			key := types.NamespacedName{Name: "pvc-zfs1"}

			By("Creating the Volume")

			vol := newZFSVolume(key.Name, zfsVolName)
			err = k8sClient.Create(ctx, vol)
			Expect(err).ToNot(HaveOccurred())

			By("Checking created ZVOL")
			Eventually(func() string {
				zvol, _ := zfs.GetDataset(zfsVolName)
				if zvol != nil {
					return zvol.Name
				}
				return ""
			}, timeout, interval).Should(Equal(zfsVolName))

			By("Deleting the Volume")
			Eventually(func() error {
				f := &zfsv1alpha1.Volume{}
				k8sClient.Get(context.Background(), key, f)
				return k8sClient.Delete(context.Background(), f)
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				f := &zfsv1alpha1.Volume{}
				return k8sClient.Get(context.Background(), key, f)
			}, timeout, interval).ShouldNot(Succeed())

			By("Checking deleted ZVOL")
			Eventually(func() error {
				_, err := zfs.GetDataset(zfsVolName)
				return err
			}, timeout, interval).Should(HaveOccurred())
		})
	})

	AfterEach(func() {
		var err error
		ctx := context.Background()
		err = k8sClient.DeleteAllOf(ctx, &zfsv1alpha1.Volume{})
		Expect(err).ToNot(HaveOccurred())
	})

})

func newZFSVolume(name, zfsVolName string) *zfsv1alpha1.Volume {
	return &zfsv1alpha1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: zfsv1alpha1.VolumeSpec{
			NodeName:   testNodeName,
			VolumeName: zfsVolName,
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}
