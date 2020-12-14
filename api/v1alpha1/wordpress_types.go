/*


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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WordpressSpec defines the desired state of Wordpress
type WordpressSpec struct {
	//Define weither or not the state should be active or archived
	// +kubebuilder:validation:Enum=Active;Archived
	// +kubebuilder:default="Archived"
	State string `json:"state"`
	//Represents the size of the MySQL and Wordpress deployments
	// +kubebuilder:validation:Enum=Small;Medium;Large
	// +kubebuilder:default="Small"
	Size string `json:"size"`
	// Tiers specify the required amount of resource that need to be bound to each pod -- Cpu / Memory
	// +kubebuilder:validation:Enum=Bronze;Silver;Gold
	// +kubebuilder:default="Bronze"
	Tier string `json:"tier"`
	// Here you can specify the Wordpress title,Admin email, Admin user, Admin password, Site URL -- Make sure to not leave the password as specified, this is just a way of setting a password for a first login.
	// +optional
	WordpressInfo WordpressInfoSpec `json:"wordpressInfo"`
}

//TODO Make Adminemail required

type WordpressInfoSpec struct {
	// The title of the Wordpress site
	// +kubebuilder:default="Default"
	// +optional
	Title string `json:"title"`
	// Defines the Wordpress URL
	// +kubebuilder:default="example.com"
	// +optional
	URL string `json:"url"`
	// Defines the Admin user
	// +kubebuilder:default="Admin"
	// +optional
	AdminUser string `json:"user"`
	// Defines the Admin password, If not set, a password will be generated
	// +kubebuilder:default=""
	// +optional
	AdminPass string `json:"password"`
	// Defines the Admin's Email
	// +kubebuilder:default="yannick.luts@hotmail.com"
	// +optional
	AdminEmail string `json:"email"`
}

// WordpressStatus defines the observed state of Wordpress
type WordpressStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//URL om naar te connecten
	URL string `json:"URL,omitempty"`
	// Generated Password for wordpress
	Password string `json:"password,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.URL",format="byte"
// +kubebuilder:printcolumn:name="Password",type="string",JSONPath=".status.password"

// Wordpress is the Schema for the wordpresses API
type Wordpress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WordpressSpec   `json:"spec,omitempty"`
	Status WordpressStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WordpressList contains a list of Wordpress
type WordpressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Wordpress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Wordpress{}, &WordpressList{})
}
