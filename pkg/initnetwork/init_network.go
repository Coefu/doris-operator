package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"time"

	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	klogv2 "k8s.io/klog/v2"
)

func main() {
	klogv2.InitFlags(nil)
	defer klogv2.Flush()

	//// outcluster
	//var kubeconfig *string
	//if home := "/"; home != "" {
	//	kubeconfig = flag.String("kubeconfig", filepath.Join(home, "", "yaml-test"), "(optional) absolute path to the kubeconfig file")
	//} else {
	//	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	//}
	//flag.Parse()
	//// use the current context in kubeconfig
	//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	//if err != nil {
	//	klogv2.Error(err.Error())
	//}

	// incluster
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		klogv2.Error(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klogv2.Error(err.Error())
	}

	// create dynamic client
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		klogv2.Error(err)
	}

	// get all apiversion name
	apiresourceslists, err := clientset.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		klogv2.Error(err.Error())
	}
	var groupVersions []string
	for _, v := range apiresourceslists {
		groupVersions = append(groupVersions, v.GroupVersion)
	}

	// init network logic
	if IsContain("crd.projectcalico.org/v1", groupVersions) {
		// get all ippools name
		ippoolsGVR := schema.GroupVersionResource{Group: "crd.projectcalico.org", Version: "v1", Resource: "ippools"}
		ippoolsList, err := dclient.Resource(ippoolsGVR).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			klogv2.Error(err)
		}
		var ippoolsNameList []string
		for _, v := range ippoolsList.Items {
			ippoolsNameList = append(ippoolsNameList, v.GetName())
		}

		if IsContain("doris-ippool", ippoolsNameList) {
			//pass
			klogv2.Info("calico api is exsited, ippools is exsited, nothing to do.")
		} else {
			//create multus CRD, multus calico NAD, calico ippool
			create(dclient, clientset, "manifest/multus-daemonset-thick-plugin.yml")
			create(dclient, clientset, "manifest/multus-calico-nad.yaml")
			create(dclient, clientset, "./manifest/calico-ippool.yaml")
		}
	} else {
		//create multus CRD,calico CRD,multus calico NAD, calico ippool
		create(dclient, clientset, "manifest/multus-daemonset-thick-plugin.yml")
		create(dclient, clientset, "manifest/calico.yaml")
		create(dclient, clientset, "manifest/multus-calico-nad.yaml")
		create(dclient, clientset, "manifest/calico-ippool.yaml")
	}
}

func IsContain(str string, array []string) bool {
	for _, v := range array {
		if v == str {
			return true
		}
	}
	return false
}

func create(dynamiClient dynamic.Interface, clientSet *kubernetes.Clientset, yamlFile string) {
	filebytes, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		klogv2.Error(err)
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(filebytes), 1024)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break //is over
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			klogv2.Warningln("yaml file has blank blocks.")
			continue //blank block
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			klogv2.Error(err)
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(clientSet.Discovery())
		if err != nil {
			klogv2.Error(err)
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			klogv2.Error(err)
		}

		var dri dynamic.ResourceInterface
		defaultNameSpace := "default"
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(defaultNameSpace)
			}
			dri = dynamiClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dynamiClient.Resource(mapping.Resource)
		}

		obj2, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			klogv2.Error(err)
		}

		klogv2.Infof("%s/%s created.", obj2.GetKind(), obj2.GetName())
		time.Sleep(500 * time.Millisecond)
	}
}
