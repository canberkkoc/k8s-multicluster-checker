package main

import (
	"context"
	"flag"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type clusterNs struct {
	deployments  []internalDeployment
	hpas         []internalHpa
	services     []internalService
	configmaps   []internalCm
	harborsecret string
	pod_count    int
	ingress      []internalIngress
}

type internalDeployment struct {
	replicas      int
	containerlist []internalContainer
}

type internalContainer struct {
	rqCpu string
	rqMem string
	lmCpu string
	lmMem string
}

type internalHpa struct {
	maxReplica        int
	minReplica        int
	cpuUtilPercentage int
}

type internalService struct {
	ports []map[string]string
}

type internalCm struct {
	cmName    string
	dataValue map[string]string
}
type internalIngress struct {
	serviceName string
	servicePort string
}

func get_values(config_file *string, ns *string, dstr *clusterNs) {

	namespace := *ns
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *config_file)

	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())

	}
	// check namespace
	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, v1.GetOptions{})

	if err != nil {
		panic(err.Error())

	}

	// Pods
	pod, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	dstr.pod_count = len(pod.Items)

	//Deployments

	deploymentsClient := clientset.AppsV1().Deployments(namespace)

	deployment_list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	for _, deployment := range deployment_list.Items {
		temp_dep := internalDeployment{int(*deployment.Spec.Replicas), []internalContainer{}}
		for _, cont := range deployment.Spec.Template.Spec.Containers {
			temp_dep.containerlist = append(temp_dep.containerlist,
				internalContainer{cont.Resources.Requests.Cpu().String(),
					cont.Resources.Requests.Memory().String(),
					cont.Resources.Limits.Cpu().String(),
					cont.Resources.Limits.Memory().String()})
		}
		dstr.deployments = append(dstr.deployments, temp_dep)
	}

	//HPA

	hpa_list, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	for _, hpa := range hpa_list.Items {
		dstr.hpas = append(dstr.hpas, internalHpa{int(hpa.Spec.MaxReplicas), int(*hpa.Spec.MinReplicas),
			int(*hpa.Spec.TargetCPUUtilizationPercentage)})
	}

	//Service
	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err.Error())
	}

	for _, s := range services.Items {
		tmp_port_list := []map[string]string{}
		for _, p := range s.Spec.Ports {
			tmp_port_list = append(tmp_port_list, map[string]string{"port_name": p.Name, "port": string(p.Port),
				"protocol": string(p.Protocol), "targer_port": p.TargetPort.String()})
		}
		dstr.services = append(dstr.services, internalService{tmp_port_list})
	}


	//ConfigMap

	cm, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, cmdata := range cm.Items {
		dstr.configmaps = append(dstr.configmaps, internalCm{cmdata.Name, cmdata.Data})
	}

	//Secrets

	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, secret := range secrets.Items {
		if secret.Name == "harborcred" {
			dstr.harborsecret = "OK"
		}
	}
}

func check_cluster_identicality(first_cluster *clusterNs, second_cluster *clusterNs) string {

	if reflect.DeepEqual(first_cluster, second_cluster) {
		return "OK"
	} else {
		if len(first_cluster.deployments) != len(second_cluster.deployments) {
			fmt.Println("DEPLOYMENTS NOT EQUAL")
			fmt.Println("First Cluster Results:")
			fmt.Println(first_cluster.deployments)
			fmt.Println("Second Cluster Results:")
			fmt.Println(second_cluster.deployments)
		} else if len(first_cluster.services) != len(second_cluster.services) {
			fmt.Println("SERVICES NOT EQUAL")
			fmt.Println("First Cluster Results:")
			fmt.Println(first_cluster.services)
			fmt.Println("Second Cluster Results:")
			fmt.Println(second_cluster.services)
		} else if len(first_cluster.configmaps) != len(second_cluster.configmaps) {
			fmt.Println("CONFIGMAPS NOT EQUAL")
			fmt.Println("First Cluster Results:")
			fmt.Println(first_cluster.configmaps)
			fmt.Println("Second Cluster Results:")
			fmt.Println(second_cluster.configmaps)
		} else if len(first_cluster.hpas) != len(second_cluster.hpas) {
			fmt.Println("HPAS NOT EQUAL")
			fmt.Println("First Cluster Results:")
			fmt.Println(first_cluster.hpas)
			fmt.Println("Second Cluster Results:")
			fmt.Println(second_cluster.hpas)
		} else if first_cluster.pod_count != second_cluster.pod_count {
			fmt.Println("POD COUNT NOT EQUAL")
			fmt.Println("First Cluster Results:")
			fmt.Println(first_cluster.pod_count)
			fmt.Println("Second Cluster Results:")
			fmt.Println(second_cluster.pod_count)
		} else if first_cluster.harborsecret != second_cluster.harborsecret {
			fmt.Println("HARBOR SECRET NOT EQUAL")
			fmt.Println("First Cluster Results:")
			fmt.Println(first_cluster.harborsecret)
			fmt.Println("Second Cluster Results:")
			fmt.Println(second_cluster.harborsecret)
		}
		return "NOT OK"
	}
}

func main() {

	var kubeconfig1, kubeconfig2, namespace *string

	kubeconfig1 = flag.String("kubeconfig1", "", "absolute path to the kubeconfig1 file")
	kubeconfig2 = flag.String("kubeconfig2", "", "absolute path to the kubeconfig2 file")
	namespace = flag.String("namespace", "", "namespace name for check")

	flag.Parse()
	n1 := &clusterNs{}
	n2 := &clusterNs{}
	get_values(kubeconfig1, namespace, n1)
	get_values(kubeconfig2, namespace, n2)
	fmt.Println(check_cluster_identicality(n1, n2))
}
