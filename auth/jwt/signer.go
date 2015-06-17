// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package jwt implements Zenoss-specific JWT facilities
package jwt

import (
	jwtgo "github.com/dgrijalva/jwt-go"
)

// A package-private facade that implements the Signer interface while hiding
// implementation details specific to the dgrijalva/jwt-go library
type signerFacade struct {
	signingMethod jwtgo.SigningMethod
}

// assert the interface
var _ Signer = &signerFacade{}

// Returns a base64-encoded signature of 'msg'.
func (facade *signerFacade) Sign(msg string, key interface{}) (string, error) {
	return facade.signingMethod.Sign(msg, key)
}

// Verifies that signed value of 'msg', matches the value of 'signature'.
// Both 'msg' and 'signature' should base-64 encoded strings.
func (facade *signerFacade) Verify(msg, signature string, key interface{}) error {
	return facade.signingMethod.Verify(msg, signature, key)
}

// Returns the JWT-standard name for the signing algorithm.
func (facade *signerFacade) Algorithm() string {
	return facade.signingMethod.Alg()
}
