// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metricbeat

import (
	"fmt"

	commonbeat "github.com/elastic/cloud-on-k8s/pkg/controller/common/beat"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/container"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/defaults"
	"github.com/elastic/cloud-on-k8s/pkg/controller/common/volume"
	corev1 "k8s.io/api/core/v1"
)

const (
	Type commonbeat.Type = "metricbeat"

	DockerSockVolumeName = "dockersock"
	DockerSockPath       = "/var/run/docker.sock"
	DockerSockMountPath  = "/var/run/docker.sock"

	ProcVolumeName = "proc"
	ProcPath       = "/proc"
	ProcMountPath  = "/hostfs/proc"

	CGroupVolumeName = "cgroup"
	CGroupPath       = "/sys/fs/cgroup"
	CGroupMountPath  = "/hostfs/sys/fs/cgroup"

	HostMetricbeatDataVolumeName   = "data"
	HostMetricbeatDataPathTemplate = "/var/lib/%s/%s/metricbeat-data"
	HostMetricbeatDataMountPath    = "/usr/share/metricbeat/data"
)

type Driver struct {
	commonbeat.DriverParams
	commonbeat.Driver
}

func NewDriver(params commonbeat.DriverParams) commonbeat.Driver {
	return &Driver{DriverParams: params}
}

func (d *Driver) Reconcile() commonbeat.DriverResults {
	f := func(builder *defaults.PodTemplateBuilder) {
		dockerSockVolume := volume.NewHostVolume(DockerSockVolumeName, DockerSockPath, DockerSockMountPath, false, corev1.HostPathUnset)
		procVolume := volume.NewReadOnlyHostVolume(ProcVolumeName, ProcPath, ProcMountPath)
		cgroupVolume := volume.NewReadOnlyHostVolume(CGroupVolumeName, CGroupPath, CGroupMountPath)

		var metricbeatDataVolume volume.VolumeLike
		if d.Deployment != nil {
			metricbeatDataVolume = volume.NewPersistentVolumeClaim(HostMetricbeatDataVolumeName, HostMetricbeatDataMountPath)
		} else {
			hostMetricbeatDataPath := fmt.Sprintf(HostMetricbeatDataPathTemplate, d.Owner.GetNamespace(), d.Owner.GetName())
			metricbeatDataVolume = volume.NewHostVolume(
				HostMetricbeatDataVolumeName,
				hostMetricbeatDataPath,
				HostMetricbeatDataMountPath,
				false,
				corev1.HostPathDirectoryOrCreate)
		}

		for _, volume := range []volume.VolumeLike{
			dockerSockVolume,
			procVolume,
			cgroupVolume,
			metricbeatDataVolume,
		} {
			builder.WithVolumes(volume.Volume()).WithVolumeMounts(volume.VolumeMount())
		}

		builder.WithArgs("-e", "-c", commonbeat.ConfigMountPath(d.Type), "-system.hostfs=/hostfs")
	}

	if d.DaemonSet == nil && d.Deployment == nil {
		d.DaemonSet = &commonbeat.DaemonSetSpec{}
	}

	return commonbeat.Reconcile(
		d.DriverParams,
		defaultConfig,
		container.MetricbeatImage,
		f)
}
