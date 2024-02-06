package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/retry"
	"os"
	"path/filepath"
)

// prompt used for see the changes of object creation
func prompt() {
	fmt.Println("-> enter any key")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("error message:")
		fmt.Println(err)
	}
}

func main() {
	var kubeconfig *string

	home := homedir.HomeDir()

	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "actual path to kubeconfigure file")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	fmt.Println(kubeconfig)

	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("error from cluster %s", err.Error())
		}
	}

	//for unstructured object need to create dynamicClient
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("error vreating from dynamic client: %v\n", err)
	}

	//group-version-resources create
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	objectDep := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "dynamic-deploy",
			},
			"spec": map[string]interface{}{
				"replicas": 4,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "demo",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "demo",
						},
					},
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  "api",
								"image": "nginx:1.12",
								"ports": []map[string]interface{}{
									{
										"name":          "http",
										"protocol":      "TCP",
										"containerPort": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	fmt.Println("Deployment Creation")
	r1, err := dynamicClient.Resource(gvr).Namespace(apiv1.NamespaceDefault).Create(ctx, objectDep, v1.CreateOptions{})
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("created deployment %q. \n", r1.GetName())
	prompt()

	///to avoid conflict
	clash := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		r1, err := dynamicClient.Resource(gvr).Namespace(apiv1.NamespaceDefault).Get(ctx, "dynamic-deploy", v1.GetOptions{})
		if err != nil {
			fmt.Println(err.Error())
		}

		err = unstructured.SetNestedField(r1.Object, int64(1), "spec", "replicas")
		if err != nil {
			fmt.Println(err.Error())
		}

		containers, found, err := unstructured.NestedSlice(r1.Object, "spec", "template", "spec", "containers")

		if err != nil || !found || containers == nil {
			fmt.Println(err.Error())
		}
		err = unstructured.SetNestedField(containers[0].(map[string]interface{}), "nginx:1.13", "image")
		if err != nil {
			//fmt.Println(err)
			panic(err)
		}
		err = unstructured.SetNestedField(r1.Object, containers, "spec", "template", "spec", "containers")
		if err != nil {
			//fmt.Println(err)
			panic(err)
		}

		_, err = dynamicClient.Resource(gvr).Namespace(apiv1.NamespaceDefault).Update(ctx, r1, v1.UpdateOptions{})
		return err

	})

	if clash != nil {
		fmt.Println(err)
	}

	fmt.Println("Updated Deployment")
	prompt()

	///listing deployments of namespaces
	fmt.Printf("deployments in Namespaces %s: ", apiv1.NamespaceDefault)

	list, err := dynamicClient.Resource(gvr).Namespace(apiv1.NamespaceDefault).List(ctx, v1.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, deploy := range list.Items {
		replicas, found, err := unstructured.NestedInt64(deploy.Object, "spec", "replicas")

		if err != nil || !found {
			fmt.Printf("Replicas not found for deployment %s: error = %s", deploy.GetName(), err)
			continue
		}
		fmt.Printf("DeploymentName: %s & have replicas: %v\n", deploy.GetName(), replicas)
	}

	prompt()

	//delete deployments
	fmt.Printf("Delete deployment: ")

	dlt := v1.DeletePropagationForeground

	if err := dynamicClient.Resource(gvr).Namespace(apiv1.NamespaceDefault).Delete(ctx, "dynamic-deploy", v1.DeleteOptions{
		PropagationPolicy: &dlt,
	}); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Deleted Deployments.")

	//for listing out the pods, from thomas stringer blog
	pods, err := dynamicClient.Resource(gvr).Namespace("kube-system").List(ctx, v1.ListOptions{})
	if err != nil {
		fmt.Printf("error getting from pod %v\n", err)
	}

	for _, pod := range pods.Items {
		fmt.Printf(
			"Name: %s\n",
			pod.Object["metadata"].(map[string]interface{})["name"])
	}

}
