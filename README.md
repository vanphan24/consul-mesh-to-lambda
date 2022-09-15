# consul-mesh-to-lambda
Demo showing service in mesh connecting to another service in a lambda function.

This repo will demo a simple two service app called Fake Service where the frontend service will connect to a backend service. The frontend service will be on an EKS cluster in Consul's service mesh. The backend service will be a lambda function.

In this demo, we will:

- Connect a Consul client cluster running on Kubernetes (EKS) to an HCP Consul cluster on AWS.
- Deploy frontend and backend service on EKS.
- Deploy a lambda function.
- Configure Consul to be aware of Lambda function.
- Point frontend service to connect to Lambda backend service on Lambda instead of local backend service on EKS.


# Pre-Req

1. This repo assume that you have already deployed an HCP Consul on AWS cluster. It is very straight forward. You can deploy one using the HCP Consul portal or via Terraform. Both options are provided from the HCP Consul portal.
  
2. Ensure you followed provided steps to peer network connections between the HCP HashiCorp Vritual network (HVN) and your own VPC.
  Note: This [Learn guide](https://learn.hashicorp.com/tutorials/cloud/consul-deploy?in=cloud/consul-cloud) can walk you through the steps of setting up HCP Consul on AWS and peering to your VPC.

3. Ensure you have followed provided steps to route traffic through your peering connections.
4. Deploy an EKS cluster in the VPC of which you've connected (peered) to HCP Consul. 
5. Ensure you have helm installed on your machine. We will use helm to install the Consul clients to your EKS clusters.
6. Ensure you ahve the [AWS CLI](https://aws.amazon.com/cli/) installed on your local machine.

# Deploy Consul Clients on AKS

Once your HCP Consul isa deployed and you've peered and routed your VPC, you can start configuring your Consul clients to be able to connect the HCP Consul Cluster.


1. Clone this repo.

```
git clone https://github.com/vanphan24/consul-mesh-to-lambda.git
```

2. Navigate to the ```/consul-mesh-to-lambda``` directory.
```
cd /consul-mesh-to-lambda
```


3. Set your environmental variables to be able to connect to your AWS account
```
export AWS_ACCESS_KEY_ID=*****************
export AWS_SECRET_ACCESS_KEY=*************
```

4. Connect your local machine's terminal to your EKS cluster.

```
aws eks --region <your-region> update-kubeconfig --name <your_eks_cluster_name>
```


6. On the HCP portal, go to your HCP Consul cluster and download the client files.  
You can click the **Access Consul** dropdown and then click **Download to install Client Agents** to download a zip archive that contains the necessary files to join your client agents to the cluster.  


![Client download](https://github.com/hashicorp/admin-partitions/blob/main/images/Screen%20Shot%202022-08-22%20at%2012.45.14%20PM.png)

7. Unzip the client config package and use **ls** to confirm that both the client_config.json and ca.pem files are available.  
  Then copy files into your ```/admin-partitions/aks/deploy-on-hcp-consul-azure-aks``` working directory.  
  

8. On the HCP portal, go to your HCP Consul cluster. 

![hcp](https://github.com/hashicorp/admin-partitions/blob/main/images/Screen%20Shot%202022-08-22%20at%201.00.26%20PM.png)

- Click on **Access Consul**. 
- Click on **Public**
- Under **Access your cluster over the public internet**, click the copy icon.  

The HCP Consul dashboard UI link is now in your clipboard. Set this UI link to the CONSUL_HTTP_ADDR environment variable on your terminal so that you can reference it later in the tutorial.  

```
export CONSUL_HTTP_ADDR=<Consul_dashboard_ui_link>
```

9. On the HCP portal, go to your HCP Consul cluster.  

![hcp-admin-token](https://github.com/hashicorp/admin-partitions/blob/main/images/Screen%20Shot%202022-08-22%20at%201.17.50%20PM.png)


- Click on **Access Consul**. 
- Select **Generate admin token** and then click the copy icon from the dialog box. 
- A global-management root token is now in your clipboard. 
 
Set this token to the CONSUL_HTTP_TOKEN environment variable on your terminal so that you can reference it later in the tutorial.

```
export CONSUL_HTTP_TOKEN=<Consul_root_token>
```

10. Use the ca.pem file in the current working directory to create a Kubernetes secret to store the Consul CA certificate. 
```
kubectl create secret generic "consul-ca-cert" --from-file='tls.crt=./ca.pem' 
```


11. The Consul gossip encryption key is embedded in the client_config.json file that you downloaded and extracted into your current directory. Issue the following command to create a Kubernetes secret that stores the Consul gossip key encryption key. The following command uses jq to extract the value from the client_config.json file.  

```
kubectl create secret generic "consul-gossip-key" --from-literal="key=$(jq -r .encrypt client_config.json)"  
```


12. The last secret you need to add is an ACL bootstrap token. You can use the one you set to your CONSUL_HTTP_TOKEN environment variable earlier. Issue the following command to create a Kubernetes secret to store the bootstrap ACL token.  

```
kubectl create secret generic "consul-bootstrap-token" --from-literal="token=${CONSUL_HTTP_TOKEN}" 
```


# Create Consul configuration files for each team

13.  Issue the following command to set the HCP Consul cluster DATACENTER environment variable, extracted from the client_config.json file. This env variable will be used in your Consul helm value file.

```
export DATACENTER=$(jq -r .datacenter client_config.json)
```

14. Extract the private server URL from the client_config.json file so that it can be set in the Helm values file as the *externalServers:hosts entry*. 
```
export RETRY_JOIN=$(jq -r --compact-output .retry_join client_config.json)
```

15. Extract the public server URL from the client_config.json file so that it can be set in the Helm values file as the **k8sAuthMethodHost** entry.

```
export KUBE_API_URL=$(kubectl config view -o jsonpath="{.clusters[?(@.name == \"$(kubectl config current-context)\")].cluster.server}")
```



16. Validate that your environment variables are correct.
```
echo $DATACENTER && \
echo $RETRY_JOIN && \
echo $KUBE_API_URL

```
The output should look similar to the following:
```
consul-cluster-demo
["servers-private-consul-f3239351.7171f9f3.z1.hashicorp.cloud"]
https://dc1-k8s-9f690a3c.hcp.westus2.azmk8s.io:443
```

17. Run the following command to generate the Helm values file. Notice the environment variables *${DATACENTER}*, *${KUBE_API_URL}*, and *${RETRY_JOIN}* will be used to reflect your specific EKS cluster values.  

Also notice ```enable_serverless_plugin``` is set to ```true```.

```
cat > config.yaml << EOF
global:
  name: consul
  enabled: false
  datacenter: ${DATACENTER}
  acls:
    manageSystemACLs: true
    bootstrapToken:
      secretName: consul-bootstrap-token
      secretKey: token
  gossipEncryption:
    secretName: consul-gossip-key
    secretKey: key
  tls:
    enabled: true
    enableAutoEncrypt: true
    caCert:
      secretName: consul-ca-cert
      secretKey: tls.crt
  enableConsulNamespaces: true
externalServers:
  enabled: true
  hosts: ${RETRY_JOIN}
  httpsPort: 443
  useSystemRoots: true
  k8sAuthMethodHost: ${KUBE_API_URL}
client:
  enabled: true
  join: ${RETRY_JOIN}
  extraConfig: |
    {
      "connect": {
        "enable_serverless_plugin": true
      }
    }
 
connectInject:
  enabled: true
  enable_serverless_plugin: true
controller:
  enabled: true
ingressGateways:
  enabled: true
  defaults:
    replicas: 1
  gateways:
    - name: ingress-gateway
      service:
        type: LoadBalancer
EOF
```

18. Deploy frontend service

```
kubectl apply -f fakeapp/frontend.yaml 
```


# Deploy Lambda Function

We will now create the backend service lambda function. 

1. Navigate to the cd ```envoy-lambda-test``` folder

```
cd envoy-lambda-test
```

2. Create a .zip file that can be uploaded to Lambda via the AWS console:

```
GOOS=linux go build main.go && zip envoy-lambda-test.zip main
```

3. Go to your AWS Console, navigate to the Lambda service console.

4. Click on **Create function**

- Give it a function name: ```backend-lambda-fakeapp```
- Select **Go 1.x** for the runtime.
- Click **Create function***=

5. Once created, in the newly created Lambda function window, click **Upload from** and select **.zip file**.

6. Upload the envoy-lambda-test.zip zip file you created earlier.

7. Once upload completes, in the same Lambda function window, click **Edit** for the **Runtime settings** box.

8. Change the Handler box to ```main```

9. Now you can manually be able to invoke the function from your terminal to test ot works.

```
aws lambda invoke --region <region-of-your-lambda-function>  --function-name backend-lambda-fakeapp --payload "$(echo '"hello"' | base64)" response.json 
```

10. The output will return in the response.json file. It will echo the string 'hello' in the above command.
```
cat response.json 
{"body":"hello","statusCode":200}%                     
```

# IAM Policy

In order for the  frontend service to be able to invoke the backend lambda functiopn, it needs to have IAM permissions to invoke the Lambda function. 
For the sake of simplicy of this demo, we will just ensure the IAM role used by the EKS work nodes have the invoke lambda permisssions. The frontend service running on the EKS worker nodes will then inherit the permissions from the nodes.

1. In your AWS Elastic Kubernetes Service (EKS) console window, go to your EKS cluster.
2. Click on Compute tab
   - Under Node group column, click on the node group
3. In Node group page, under the Details tab, there’s an **Node IAM role ARN** which refers to the IAM role the node group is using.
   - Click on the ARN role.
4. It takes you to the IAM role page for thie IAM role. Click on **Add permission->Attach policy**
5. Click on **Add Permissions** and select **Create Inline**
6. Click on JSON tab and copy the following permissions in the box:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "Invoke",
            "Effect": "Allow",
            "Action": [
                "lambda:InvokeFunction"
            ],
            "Resource": "<Your-Lambda-ARN>"
        }
    ]
}
```
Note, instead of <Your-Lambda-ARN>, you can set it to "*" which will give permission to invoke ***any*** lambda function.
7. Click **Review policy**, provide name for new policy, and **Create policy**.


# Configure Lambda on Consul

Next, we need to register our lambda function to Consul so Consul knows about it. We will do it the manual way but there’s an automated registration method using Terraform to deploy a [Lambda registrator](https://www.consul.io/docs/lambda/registration#automatic-lambda-function-registration).


1. 







