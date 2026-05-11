//go:build !app_engine && as_proxy

// Copyright 2014-2022 Aerospike, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aerospike

// QueryAggregate executes a Map/Reduce query and returns the results.
// The query executor puts records on the channel from separate goroutines.
// The caller can concurrently pop records off the channel through the
// Recordset.Records channel.
//
// This method is only supported by Aerospike 3+ servers.
// If the policy is nil, the default relevant policy will be used.
func (clnt *ProxyClient) QueryAggregate(policy *QueryPolicy, statement *Statement, packageName, functionName string, functionArgs ...Value) (*Recordset, Error) {
	panic("NOT SUPPORTED")
}
