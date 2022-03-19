# kube-events

Sample code to demonstrate watches on any resource including CRDs using dynamic types. 

Each watch can monitor a resource in a namespace, a resource type in a namespace, or a resource type across all namespaces.

Uses a in-memory cache to keep recently changed objects and calculates a patch.

The intended usage is when `informers` are not an option due to cache scalability.

