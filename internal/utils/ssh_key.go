/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
)

func GenerateKey(bits int) (*rsa.PrivateKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil,err
	}
	return private, nil
}

func MakeSSHKeyPair() (privateKey string,pubKey string,err error) {
	var rsaKey * rsa.PrivateKey
	if rsaKey,  err = GenerateKey(2048);err != nil {
         return
	}
	if sshPub,fail := ssh.NewPublicKey(&rsaKey.PublicKey);fail != nil {
		err = fail
		return
	}else{
		pubKey=string(ssh.MarshalAuthorizedKey(sshPub))
	}
	privateKey = string(pem.EncodeToMemory(&pem.Block{
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		Type:  "RSA PRIVATE KEY",
	}))
	return
}

