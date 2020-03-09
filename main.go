package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the namespace as the first argument")
	}

	namespace := os.Args[1]
	fmt.Printf("Tailing pods in namespace %s\n", namespace)

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the namespace %s\n", len(pods.Items), namespace)

	var wg sync.WaitGroup
	for _, pod := range pods.Items {
		wg.Add(1)
		go echoPodLogs(clientset, namespace, pod.Name, &wg)
	}

	wg.Wait()
}

func echoPodLogs(clientset *kubernetes.Clientset, namespace string, podName string, wg *sync.WaitGroup) {
	defer wg.Done()

	sinceSeconds := int64(30)
	plo := &v1.PodLogOptions{
		Follow:       true,
		SinceSeconds: &sinceSeconds,
	}
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, plo)
	podLogs, err := req.Stream()
	if err != nil {
		panic("error in opening string")
	}
	defer podLogs.Close()

	for {
		buf := make([]byte, 256)
		n, err := podLogs.Read(buf)
		if err != nil && err == io.EOF {
			fmt.Println("EOF... done")
			break
		}

		if n == 0 {
			continue
		}

		fmt.Printf("[%s] %s\n", podName, string(buf[0:n]))
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
