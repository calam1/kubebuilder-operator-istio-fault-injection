package faultinjection

import (
	"context"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/protobuf/types/known/structpb"
	resiliencyv1 "grainger.com/api/v1"
	"istio.io/api/networking/v1alpha3"
	clientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateFaultInjectionEnvoyFilter(instance *resiliencyv1.FaultInjection) *clientnetworking.EnvoyFilter {
	reqLogger := log.FromContext(context.TODO())
	reqLogger.Info("=== Getting Fault Injection Envoy Filter Creating it")

	return &clientnetworking.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Spec.Name,
			Namespace: instance.Spec.Namespace,
		},
		Spec: v1alpha3.EnvoyFilter{
			WorkloadSelector: &v1alpha3.WorkloadSelector{
				Labels: map[string]string{
					"app": "python-api",
				},
			},
			ConfigPatches: []*v1alpha3.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: v1alpha3.EnvoyFilter_HTTP_FILTER,
					Match: &v1alpha3.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: v1alpha3.EnvoyFilter_SIDECAR_INBOUND,
						ObjectTypes: &v1alpha3.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &v1alpha3.EnvoyFilter_ListenerMatch{
								FilterChain: &v1alpha3.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &v1alpha3.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
									},
								},
							},
						},
					},

					Patch: &v1alpha3.EnvoyFilter_Patch{
						Operation: v1alpha3.EnvoyFilter_Patch_INSERT_BEFORE,
						Value: &_struct.Struct{
							Fields: map[string]*structpb.Value{
								"name": {
									Kind: &structpb.Value_StringValue{
										StringValue: "envoy.fault",
									},
								},
								"typed_config": {
									Kind: &structpb.Value_StructValue{
										StructValue: &_struct.Struct{
											Fields: map[string]*structpb.Value{
												"@type": {
													Kind: &structpb.Value_StringValue{
														StringValue: "type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault",
													},
												},
												"abort": {
													Kind: &structpb.Value_StructValue{
														StructValue: &_struct.Struct{
															Fields: map[string]*structpb.Value{
																"header_abort": {
																	Kind: &structpb.Value_StructValue{
																		StructValue: &_struct.Struct{},
																	},
																},
																"percentage": {
																	Kind: &structpb.Value_StructValue{
																		StructValue: &_struct.Struct{
																			Fields: map[string]*structpb.Value{
																				"numerator": {
																					Kind: &structpb.Value_NumberValue{
																						NumberValue: 100,
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

}

// apiVersion: networking.istio.io/v1alpha3
// kind: EnvoyFilter
// metadata:
//   name: python-api-filter
// spec:
//   workloadSelector:
//     labels:
//       app: python-api
//   configPatches:
//   - applyTo: HTTP_FILTER
//     match:
//       context: SIDECAR_INBOUND
//       listener:
//         filterChain:
//           filter:
//             name: "envoy.filters.network.http_connection_manager"
//     patch:
//       operation: INSERT_BEFORE
//       value:
//         name: envoy.fault
//         typed_config:
//           "@type": "type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault"
//           abort:
//             header_abort: {}
//             percentage:
//               numerator: 100
//           delay:
//             header_delay: {}
//             percentage:
//               numerator: 100
//           response_rate_limit:
//             header_limit: {}
//             percentage:
//               numerator: 100
