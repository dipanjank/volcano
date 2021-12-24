/*
Copyright 2020 The Volcano Authors.

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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto" // auto-registry collectors in default registry
)

var (
	queueAllocatedMilliCPU = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_allocated_milli_cpu",
			Help:      "Allocated CPU count for one queue",
		}, []string{"queue_name"},
	)

	queueAllocatedMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_allocated_memory_bytes",
			Help:      "Allocated memory for one queue",
		}, []string{"queue_name"},
	)

	queueRequestMilliCPU = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_request_milli_cpu",
			Help:      "Request CPU count for one queue",
		}, []string{"queue_name"},
	)

	queueRequestMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_request_memory_bytes",
			Help:      "Request memory for one queue",
		}, []string{"queue_name"},
	)

	queueTotalMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_total_memory_bytes",
			Help:      "Total memory for one queue",
		}, []string{"queue_name"},
	)

	queueTotalMilliCPU = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_total_milli_cpu",
			Help:      "Total CPU count for one queue",
		}, []string{"queue_name"},
	)

	queueDeservedMilliCPU = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_deserved_milli_cpu",
			Help:      "Deserved CPU count for one queue",
		}, []string{"queue_name"},
	)

	queueDeservedMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_deserved_memory_bytes",
			Help:      "Deserved memory for one queue",
		}, []string{"queue_name"},
	)

	queuePreemptibleMilliCPU = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_preemptible_milli_cpu",
			Help:      "Preemptible CPU count for one queue",
		}, []string{"queue_name"},
	)

	queuePreemptibleMemory = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_preemptible_memory_bytes",
			Help:      "Preemptible memory for one queue",
		}, []string{"queue_name"},
	)

	queueShare = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_share",
			Help:      "Share for one queue",
		}, []string{"queue_name"},
	)

	queueDeservedShare = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_deserved_share",
			Help:      "Rest share for one queue",
		}, []string{"queue_name"},
	)

	queueWeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_weight",
			Help:      "Weight for one queue",
		}, []string{"queue_name"},
	)

	queueHierarchyWeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_hierarchy_weight",
			Help:      "Hierarchy weight for one queue",
		}, []string{"queue_name"},
	)

	queueTotalHierarchyWeights = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_total_hierarchy_weights",
			Help:      "Total hierarchy weight for one queue",
		}, []string{"queue_name"},
	)

	queueOverused = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_overused",
			Help:      "If one queue is overused",
		}, []string{"queue_name"},
	)

	queuePodGroupInqueue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_pod_group_inqueue_count",
			Help:      "The number of Inqueue PodGroup in this queue",
		}, []string{"queue_name"},
	)

	queuePodGroupPending = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_pod_group_pending_count",
			Help:      "The number of Pending PodGroup in this queue",
		}, []string{"queue_name"},
	)

	queuePodGroupRunning = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_pod_group_running_count",
			Help:      "The number of Running PodGroup in this queue",
		}, []string{"queue_name"},
	)

	queuePodGroupUnknown = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_pod_group_unknown_count",
			Help:      "The number of Unknown PodGroup in this queue",
		}, []string{"queue_name"},
	)

	queueActiveJobs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: VolcanoNamespace,
			Name:      "queue_active_jobs",
			Help:      "The number of acive jobs in this queue",
		},
		[]string{"queue_name"},
	)
)

// UpdateQueueAllocated records allocated resources for one queue
func UpdateQueueAllocated(queueName string, milliCPU, memory float64) {
	queueAllocatedMilliCPU.WithLabelValues(queueName).Set(milliCPU)
	queueAllocatedMemory.WithLabelValues(queueName).Set(memory)
}

// UpdateQueueRequest records request resources for one queue
func UpdateQueueRequest(queueName string, milliCPU, memory float64) {
	queueRequestMilliCPU.WithLabelValues(queueName).Set(milliCPU)
	queueRequestMemory.WithLabelValues(queueName).Set(memory)
}

// UpdateQueueDeserved records deserved resources for one queue
func UpdateQueueDeserved(queueName string, milliCPU, memory float64) {
	queueDeservedMilliCPU.WithLabelValues(queueName).Set(milliCPU)
	queueDeservedMemory.WithLabelValues(queueName).Set(memory)
}

// UpdateQueueTotalAllocatable records the total deserved resources for one queue
func UpdateQueueTotalAllocatable(queueName string, milliCPU, memory float64) {
	queueTotalMilliCPU.WithLabelValues(queueName).Set(milliCPU)
	queueTotalMemory.WithLabelValues(queueName).Set(memory)
}

// UpdateQueuePreemptible records deserved resources for one queue
func UpdateQueuePreemptible(queueName string, milliCPU, memory float64) {
	queuePreemptibleMilliCPU.WithLabelValues(queueName).Set(milliCPU)
	queuePreemptibleMemory.WithLabelValues(queueName).Set(memory)
}

// UpdateQueueShare records share for one queue
func UpdateQueueShare(queueName string, share float64) {
	queueShare.WithLabelValues(queueName).Set(share)
}

// UpdateQueueDeservedShare records deserved share for one queue
func UpdateQueueDeservedShare(queueName string, deservedShare float64) {
	queueDeservedShare.WithLabelValues(queueName).Set(deservedShare)
}

// UpdateQueueWeight records weight for one queue
func UpdateQueueWeight(queueName string, weight int32) {
	queueWeight.WithLabelValues(queueName).Set(float64(weight))
}

// UpdateQueueHierarchyWeight records hierarchical weight for one queue used for the drf plugin
func UpdateQueueHierarchyWeight(queueName string, weight float64) {
	queueHierarchyWeight.WithLabelValues(queueName).Set(weight)
}

// UpdateQueueTotalHierarchyWeights records total hierarchical weights of the parent queue
func UpdateQueueTotalHierarchyWeights(queueName string, total float64) {
	queueTotalHierarchyWeights.WithLabelValues(queueName).Set(total)
}

// UpdateQueueOverused records if one queue is overused
func UpdateQueueOverused(queueName string, overused bool) {
	var value float64
	if overused {
		value = 1
	} else {
		value = 0
	}
	queueOverused.WithLabelValues(queueName).Set(value)
}

// UpdateQueuePodGroupInqueueCount records the number of Inqueue PodGroup in this queue
func UpdateQueuePodGroupInqueueCount(queueName string, count int32) {
	queuePodGroupInqueue.WithLabelValues(queueName).Set(float64(count))
}

// UpdateQueuePodGroupPendingCount records the number of Pending PodGroup in this queue
func UpdateQueuePodGroupPendingCount(queueName string, count int32) {
	queuePodGroupPending.WithLabelValues(queueName).Set(float64(count))
}

// UpdateQueuePodGroupRunningCount records the number of Running PodGroup in this queue
func UpdateQueuePodGroupRunningCount(queueName string, count int32) {
	queuePodGroupRunning.WithLabelValues(queueName).Set(float64(count))
}

// UpdateQueuePodGroupUnknownCount records the number of Unknown PodGroup in this queue
func UpdateQueuePodGroupUnknownCount(queueName string, count int32) {
	queuePodGroupUnknown.WithLabelValues(queueName).Set(float64(count))
}

// UpdateQueuePodGroupUnknownCount records the number of Unknown PodGroup in this queue
func UpdateQueueActiveJobs(queueName string, count float64) {
	queueActiveJobs.WithLabelValues(queueName).Set(count)
}
