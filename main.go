package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	_ "k8s.io/client-go/util/retry"
	"path/filepath"
)

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

	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("error from cluster %s", err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	//deploymentClient
	dep := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name: "apiserver-deploy",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(3),
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "api",
							Image: "takia111/new-image",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
							Args: []string{
								"server",
								"-p",
								"8080",
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	///deployment-create
	fmt.Println("creating deployments: ")

	r1, err := dep.Create(ctx, deployment, v1.CreateOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Created deployments are:%q. \n", r1.GetObjectMeta().GetName())
	prompt()

	fmt.Println("Updating deployments are:")

	clash := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		r1, err := dep.Get(ctx, "apiserver-deploy", v1.GetOptions{})
		if err != nil {
			fmt.Println(err)
		}

		r1.Spec.Replicas = pointer.Int32Ptr(2)
		r1.Spec.Template.Spec.Containers[0].Image = "takia111/new-image"
		_, err = dep.Update(ctx, r1, v1.UpdateOptions{})
		return err
	})

	if clash != nil {
		fmt.Println(err)
		return
	}

	prompt()
	///deployment update
	fmt.Printf("listed deployments are : %s : ", apiv1.NamespaceDefault)

	list, err := dep.List(ctx, v1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, deploy := range list.Items {
		fmt.Println(deploy.Name, *deploy.Spec.Replicas)
	}
	prompt()

	//delete deployments
	fmt.Printf("Delete deployment: ")
	dlt := v1.DeletePropagationForeground
	if err := dep.Delete(ctx, "apiserver-deploy", v1.DeleteOptions{
		PropagationPolicy: &dlt,
	}); err != nil {
		fmt.Println(err)
	}

}
