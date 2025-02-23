package openpgp

import (
	"bytes"
	"crypto"
	"strings"
	"testing"
	"time"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaPHkZLbQQbvcyavn8q1GFHg6o6yeceyHFSJ3Pjf3p3TQ/go-crypto/openpgp/errors"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmaPHkZLbQQbvcyavn8q1GFHg6o6yeceyHFSJ3Pjf3p3TQ/go-crypto/openpgp/packet"
)

func TestKeyExpiry(t *testing.T) {
	kring, _ := ReadKeyRing(readerFromHex(expiringKeyHex))
	entity := kring[0]

	const timeFormat = "2006-01-02"
	time1, _ := time.Parse(timeFormat, "2013-07-01")

	// The expiringKeyHex key is structured as:
	//
	// pub  1024R/5E237D8C  created: 2013-07-01                      expires: 2013-07-31  usage: SC
	// sub  1024R/1ABB25A0  created: 2013-07-01 23:11:07 +0200 CEST  expires: 2013-07-08  usage: E
	// sub  1024R/96A672F5  created: 2013-07-01 23:11:23 +0200 CEST  expires: 2013-07-31  usage: E
	//
	// So this should select the newest, non-expired encryption key.
	key, _ := entity.encryptionKey(time1)
	if id := key.PublicKey.KeyIdShortString(); id != "96A672F5" {
		t.Errorf("Expected key 1ABB25A0 at time %s, but got key %s", time1.Format(timeFormat), id)
	}

	// Once the first encryption subkey has expired, the second should be
	// selected.
	time2, _ := time.Parse(timeFormat, "2013-07-09")
	key, _ = entity.encryptionKey(time2)
	if id := key.PublicKey.KeyIdShortString(); id != "96A672F5" {
		t.Errorf("Expected key 96A672F5 at time %s, but got key %s", time2.Format(timeFormat), id)
	}

	// Once all the keys have expired, nothing should be returned.
	time3, _ := time.Parse(timeFormat, "2013-08-01")
	if key, ok := entity.encryptionKey(time3); ok {
		t.Errorf("Expected no key at time %s, but got key %s", time3.Format(timeFormat), key.PublicKey.KeyIdShortString())
	}
}

func TestMissingCrossSignature(t *testing.T) {
	// This public key has a signing subkey, but the subkey does not
	// contain a cross-signature.
	keys, err := ReadArmoredKeyRing(bytes.NewBufferString(missingCrossSignatureKey))
	if len(keys) != 0 {
		t.Errorf("Accepted key with missing cross signature")
	}
	if err == nil {
		t.Fatal("Failed to detect error in keyring with missing cross signature")
	}
	structural, ok := err.(errors.StructuralError)
	if !ok {
		t.Fatalf("Unexpected class of error: %T. Wanted StructuralError", err)
	}
	const expectedMsg = "signing subkey is missing cross-signature"
	if !strings.Contains(string(structural), expectedMsg) {
		t.Fatalf("Unexpected error: %q. Expected it to contain %q", err, expectedMsg)
	}
}

func TestInvalidCrossSignature(t *testing.T) {
	// This public key has a signing subkey, and the subkey has an
	// embedded cross-signature. However, the cross-signature does
	// not correctly validate over the primary and subkey.
	keys, err := ReadArmoredKeyRing(bytes.NewBufferString(invalidCrossSignatureKey))
	if len(keys) != 0 {
		t.Errorf("Accepted key with invalid cross signature")
	}
	if err == nil {
		t.Fatal("Failed to detect error in keyring with an invalid cross signature")
	}
	structural, ok := err.(errors.StructuralError)
	if !ok {
		t.Fatalf("Unexpected class of error: %T. Wanted StructuralError", err)
	}
	const expectedMsg = "subkey signature invalid"
	if !strings.Contains(string(structural), expectedMsg) {
		t.Fatalf("Unexpected error: %q. Expected it to contain %q", err, expectedMsg)
	}
}

func TestGoodCrossSignature(t *testing.T) {
	// This public key has a signing subkey, and the subkey has an
	// embedded cross-signature which correctly validates over the
	// primary and subkey.
	keys, err := ReadArmoredKeyRing(bytes.NewBufferString(goodCrossSignatureKey))
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Errorf("Failed to accept key with good cross signature, %d", len(keys))
	}
	if len(keys[0].Subkeys) != 1 {
		t.Errorf("Failed to accept good subkey, %d", len(keys[0].Subkeys))
	}
}

// TestExternallyRevokableKey attempts to load and parse a key with a third party revocation permission.
func TestExternallyRevocableKey(t *testing.T) {
	kring, _ := ReadKeyRing(readerFromHex(subkeyUsageHex))

	// The 0xA42704B92866382A key can be revoked by 0xBE3893CB843D0FE70C
	// according to this signature that appears within the key:
	// :signature packet: algo 1, keyid A42704B92866382A
	//    version 4, created 1396409682, md5len 0, sigclass 0x1f
	//    digest algo 2, begin of digest a9 84
	//    hashed subpkt 2 len 4 (sig created 2014-04-02)
	//    hashed subpkt 12 len 22 (revocation key: c=80 a=1 f=CE094AA433F7040BB2DDF0BE3893CB843D0FE70C)
	//    hashed subpkt 7 len 1 (not revocable)
	//    subpkt 16 len 8 (issuer key ID A42704B92866382A)
	//    data: [1024 bits]

	id := uint64(0xA42704B92866382A)
	keys := kring.KeysById(id)
	if len(keys) != 1 {
		t.Errorf("Expected to find key id %X, but got %d matches", id, len(keys))
	}
}

func TestKeyRevocation(t *testing.T) {
	kring, _ := ReadKeyRing(readerFromHex(revokedKeyHex))

	// revokedKeyHex contains these keys:
	// pub   1024R/9A34F7C0 2014-03-25 [revoked: 2014-03-25]
	// sub   1024R/1BA3CD60 2014-03-25 [revoked: 2014-03-25]
	ids := []uint64{0xA401D9F09A34F7C0, 0x5CD3BE0A1BA3CD60}

	for _, id := range ids {
		keys := kring.KeysById(id)
		if len(keys) != 1 {
			t.Errorf("Expected KeysById to find revoked key %X, but got %d matches", id, len(keys))
		}
		keys = kring.KeysByIdUsage(id, 0)
		if len(keys) != 0 {
			t.Errorf("Expected KeysByIdUsage to filter out revoked key %X, but got %d matches", id, len(keys))
		}
	}
}

func TestSubkeyRevocation(t *testing.T) {
	kring, _ := ReadKeyRing(readerFromHex(revokedSubkeyHex))

	// revokedSubkeyHex contains these keys:
	// pub   1024R/4EF7E4BECCDE97F0 2014-03-25
	// sub   1024R/D63636E2B96AE423 2014-03-25
	// sub   1024D/DBCE4EE19529437F 2014-03-25
	// sub   1024R/677815E371C2FD23 2014-03-25 [revoked: 2014-03-25]
	validKeys := []uint64{0x4EF7E4BECCDE97F0, 0xD63636E2B96AE423, 0xDBCE4EE19529437F}
	revokedKey := uint64(0x677815E371C2FD23)

	for _, id := range validKeys {
		keys := kring.KeysById(id)
		if len(keys) != 1 {
			t.Errorf("Expected KeysById to find key %X, but got %d matches", id, len(keys))
		}
		keys = kring.KeysByIdUsage(id, 0)
		if len(keys) != 1 {
			t.Errorf("Expected KeysByIdUsage to find key %X, but got %d matches", id, len(keys))
		}
	}

	keys := kring.KeysById(revokedKey)
	if len(keys) != 1 {
		t.Errorf("Expected KeysById to find key %X, but got %d matches", revokedKey, len(keys))
	}

	keys = kring.KeysByIdUsage(revokedKey, 0)
	if len(keys) != 0 {
		t.Errorf("Expected KeysByIdUsage to filter out revoked key %X, but got %d matches", revokedKey, len(keys))
	}
}

func TestKeyUsage(t *testing.T) {
	kring, _ := ReadKeyRing(readerFromHex(subkeyUsageHex))

	// subkeyUsageHex contains these keys:
	// pub  1024R/2866382A  created: 2014-04-01  expires: never       usage: SC
	// sub  1024R/936C9153  created: 2014-04-01  expires: never       usage: E
	// sub  1024R/64D5F5BB  created: 2014-04-02  expires: never       usage: E
	// sub  1024D/BC0BA992  created: 2014-04-02  expires: never       usage: S
	certifiers := []uint64{0xA42704B92866382A}
	signers := []uint64{0xA42704B92866382A, 0x42CE2C64BC0BA992}
	encrypters := []uint64{0x09C0C7D9936C9153, 0xC104E98664D5F5BB}

	for _, id := range certifiers {
		keys := kring.KeysByIdUsage(id, packet.KeyFlagCertify)
		if len(keys) == 1 {
			if keys[0].PublicKey.KeyId != id {
				t.Errorf("Expected to find certifier key id %X, but got %X", id, keys[0].PublicKey.KeyId)
			}
		} else {
			t.Errorf("Expected one match for certifier key id %X, but got %d matches", id, len(keys))
		}
	}

	for _, id := range signers {
		keys := kring.KeysByIdUsage(id, packet.KeyFlagSign)
		if len(keys) == 1 {
			if keys[0].PublicKey.KeyId != id {
				t.Errorf("Expected to find signing key id %X, but got %X", id, keys[0].PublicKey.KeyId)
			}
		} else {
			t.Errorf("Expected one match for signing key id %X, but got %d matches", id, len(keys))
		}

		// This keyring contains no encryption keys that are also good for signing.
		keys = kring.KeysByIdUsage(id, packet.KeyFlagEncryptStorage|packet.KeyFlagEncryptCommunications)
		if len(keys) != 0 {
			t.Errorf("Unexpected match for encryption key id %X", id)
		}
	}

	for _, id := range encrypters {
		keys := kring.KeysByIdUsage(id, packet.KeyFlagEncryptStorage|packet.KeyFlagEncryptCommunications)
		if len(keys) == 1 {
			if keys[0].PublicKey.KeyId != id {
				t.Errorf("Expected to find encryption key id %X, but got %X", id, keys[0].PublicKey.KeyId)
			}
		} else {
			t.Errorf("Expected one match for encryption key id %X, but got %d matches", id, len(keys))
		}

		// This keyring contains no encryption keys that are also good for signing.
		keys = kring.KeysByIdUsage(id, packet.KeyFlagSign)
		if len(keys) != 0 {
			t.Errorf("Unexpected match for signing key id %X", id)
		}
	}
}

func TestIdVerification(t *testing.T) {
	kring, err := ReadKeyRing(readerFromHex(testKeys1And2PrivateHex))
	if err != nil {
		t.Fatal(err)
	}
	if err := kring[1].PrivateKey.Decrypt([]byte("passphrase")); err != nil {
		t.Fatal(err)
	}

	const identity = "Test Key 1 (RSA)"
	if err := kring[0].SignIdentity(identity, kring[1], nil); err != nil {
		t.Fatal(err)
	}

	ident, ok := kring[0].Identities[identity]
	if !ok {
		t.Fatal("identity missing from key after signing")
	}

	checked := false
	for _, sig := range ident.Signatures {
		if sig.IssuerKeyId == nil || *sig.IssuerKeyId != kring[1].PrimaryKey.KeyId {
			continue
		}

		if err := kring[1].PrimaryKey.VerifyUserIdSignature(identity, kring[0].PrimaryKey, sig); err != nil {
			t.Fatalf("error verifying new identity signature: %s", err)
		}
		checked = true
		break
	}

	if !checked {
		t.Fatal("didn't find identity signature in Entity")
	}
}

func TestNewEntityWithPreferredHash(t *testing.T) {
	c := &packet.Config{
		DefaultHash: crypto.SHA256,
	}
	entity, err := NewEntity("Golang Gopher", "Test Key", "no-reply@golang.com", c)
	if err != nil {
		t.Fatal(err)
	}

	for _, identity := range entity.Identities {
		if len(identity.SelfSignature.PreferredHash) == 0 {
			t.Fatal("didn't find a preferred hash in self signature")
		}
		ph := hashToHashId(c.DefaultHash)
		if identity.SelfSignature.PreferredHash[0] != ph {
			t.Fatalf("Expected preferred hash to be %d, got %d", ph, identity.SelfSignature.PreferredHash[0])
		}
	}
}

func TestNewEntityWithoutPreferredHash(t *testing.T) {
	entity, err := NewEntity("Golang Gopher", "Test Key", "no-reply@golang.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, identity := range entity.Identities {
		if len(identity.SelfSignature.PreferredHash) != 0 {
			t.Fatal("Expected preferred hash to be empty but got length %d", len(identity.SelfSignature.PreferredHash))
		}
	}
}

const expiringKeyHex = "988d0451d1ec5d010400ba3385721f2dc3f4ab096b2ee867ab77213f0a27a8538441c35d2fa225b08798a1439a66a5150e6bdc3f40f5d28d588c712394c632b6299f77db8c0d48d37903fb72ebd794d61be6aa774688839e5fdecfe06b2684cc115d240c98c66cb1ef22ae84e3aa0c2b0c28665c1e7d4d044e7f270706193f5223c8d44e0d70b7b8da830011010001b40f4578706972792074657374206b657988be041301020028050251d1ec5d021b03050900278d00060b090807030206150802090a0b0416020301021e01021780000a091072589ad75e237d8c033503fd10506d72837834eb7f994117740723adc39227104b0d326a1161871c0b415d25b4aedef946ca77ea4c05af9c22b32cf98be86ab890111fced1ee3f75e87b7cc3c00dc63bbc85dfab91c0dc2ad9de2c4d13a34659333a85c6acc1a669c5e1d6cecb0cf1e56c10e72d855ae177ddc9e766f9b2dda57ccbb75f57156438bbdb4e42b88d0451d1ec5d0104009c64906559866c5cb61578f5846a94fcee142a489c9b41e67b12bb54cfe86eb9bc8566460f9a720cb00d6526fbccfd4f552071a8e3f7744b1882d01036d811ee5a3fb91a1c568055758f43ba5d2c6a9676b012f3a1a89e47bbf624f1ad571b208f3cc6224eb378f1645dd3d47584463f9eadeacfd1ce6f813064fbfdcc4b5a53001101000188a504180102000f021b0c050251d1f06b050900093e89000a091072589ad75e237d8c20e00400ab8310a41461425b37889c4da28129b5fae6084fafbc0a47dd1adc74a264c6e9c9cc125f40462ee1433072a58384daef88c961c390ed06426a81b464a53194c4e291ddd7e2e2ba3efced01537d713bd111f48437bde2363446200995e8e0d4e528dda377fd1e8f8ede9c8e2198b393bd86852ce7457a7e3daf74d510461a5b77b88d0451d1ece8010400b3a519f83ab0010307e83bca895170acce8964a044190a2b368892f7a244758d9fc193482648acb1fb9780d28cc22d171931f38bb40279389fc9bf2110876d4f3db4fcfb13f22f7083877fe56592b3b65251312c36f83ffcb6d313c6a17f197dd471f0712aad15a8537b435a92471ba2e5b0c72a6c72536c3b567c558d7b6051001101000188a504180102000f021b0c050251d1f07b050900279091000a091072589ad75e237d8ce69e03fe286026afacf7c97ee20673864d4459a2240b5655219950643c7dba0ac384b1d4359c67805b21d98211f7b09c2a0ccf6410c8c04d4ff4a51293725d8d6570d9d8bb0e10c07d22357caeb49626df99c180be02d77d1fe8ed25e7a54481237646083a9f89a11566cd20b9e995b1487c5f9e02aeb434f3a1897cd416dd0a87861838da3e9e"
const subkeyUsageHex = "988d04533a52bc010400d26af43085558f65b9e7dbc90cb9238015259aed5e954637adcfa2181548b2d0b60c65f1f42ec5081cbf1bc0a8aa4900acfb77070837c58f26012fbce297d70afe96e759ad63531f0037538e70dbf8e384569b9720d99d8eb39d8d0a2947233ed242436cb6ac7dfe74123354b3d0119b5c235d3dd9c9d6c004f8ffaf67ad8583001101000188b7041f010200210502533b8552170c8001ce094aa433f7040bb2ddf0be3893cb843d0fe70c020700000a0910a42704b92866382aa98404009d63d916a27543da4221c60087c33f1c44bec9998c5438018ed370cca4962876c748e94b73eb39c58eb698063f3fd6346d58dd2a11c0247934c4a9d71f24754f7468f96fb24c3e791dd2392b62f626148ad724189498cbf993db2df7c0cdc2d677c35da0f16cb16c9ce7c33b4de65a4a91b1d21a130ae9cc26067718910ef8e2b417556d627261203c756d627261407379642e65642e61753e88b80413010200220502533a52bc021b03060b090807030206150802090a0b0416020301021e01021780000a0910a42704b92866382a47840400c0c2bd04f5fca586de408b395b3c280a278259c93eaaa8b79a53b97003f8ed502a8a00446dd9947fb462677e4fcac0dac2f0701847d15130aadb6cd9e0705ea0cf5f92f129136c7be21a718d46c8e641eb7f044f2adae573e11ae423a0a9ca51324f03a8a2f34b91fa40c3cc764bee4dccadedb54c768ba0469b683ea53f1c29b88d04533a52bc01040099c92a5d6f8b744224da27bc2369127c35269b58bec179de6bbc038f749344222f85a31933224f26b70243c4e4b2d242f0c4777eaef7b5502f9dad6d8bf3aaeb471210674b74de2d7078af497d55f5cdad97c7bedfbc1b41e8065a97c9c3d344b21fc81d27723af8e374bc595da26ea242dccb6ae497be26eea57e563ed517e90011010001889f0418010200090502533a52bc021b0c000a0910a42704b92866382afa1403ff70284c2de8a043ff51d8d29772602fa98009b7861c540535f874f2c230af8caf5638151a636b21f8255003997ccd29747fdd06777bb24f9593bd7d98a3e887689bf902f999915fcc94625ae487e5d13e6616f89090ebc4fdc7eb5cad8943e4056995bb61c6af37f8043016876a958ec7ebf39c43d20d53b7f546cfa83e8d2604b88d04533b8283010400c0b529316dbdf58b4c54461e7e669dc11c09eb7f73819f178ccd4177b9182b91d138605fcf1e463262fabefa73f94a52b5e15d1904635541c7ea540f07050ce0fb51b73e6f88644cec86e91107c957a114f69554548a85295d2b70bd0b203992f76eb5d493d86d9eabcaa7ef3fc7db7e458438db3fcdb0ca1cc97c638439a9170011010001889f0418010200090502533b8283021b0c000a0910a42704b92866382adc6d0400cfff6258485a21675adb7a811c3e19ebca18851533f75a7ba317950b9997fda8d1a4c8c76505c08c04b6c2cc31dc704d33da36a21273f2b388a1a706f7c3378b66d887197a525936ed9a69acb57fe7f718133da85ec742001c5d1864e9c6c8ea1b94f1c3759cebfd93b18606066c063a63be86085b7e37bdbc65f9a915bf084bb901a204533b85cd110400aed3d2c52af2b38b5b67904b0ef73d6dd7aef86adb770e2b153cd22489654dcc91730892087bb9856ae2d9f7ed1eb48f214243fe86bfe87b349ebd7c30e630e49c07b21fdabf78b7a95c8b7f969e97e3d33f2e074c63552ba64a2ded7badc05ce0ea2be6d53485f6900c7860c7aa76560376ce963d7271b9b54638a4028b573f00a0d8854bfcdb04986141568046202192263b9b67350400aaa1049dbc7943141ef590a70dcb028d730371d92ea4863de715f7f0f16d168bd3dc266c2450457d46dcbbf0b071547e5fbee7700a820c3750b236335d8d5848adb3c0da010e998908dfd93d961480084f3aea20b247034f8988eccb5546efaa35a92d0451df3aaf1aee5aa36a4c4d462c760ecd9cebcabfbe1412b1f21450f203fd126687cd486496e971a87fd9e1a8a765fe654baa219a6871ab97768596ab05c26c1aeea8f1a2c72395a58dbc12ef9640d2b95784e974a4d2d5a9b17c25fedacfe551bda52602de8f6d2e48443f5dd1a2a2a8e6a5e70ecdb88cd6e766ad9745c7ee91d78cc55c3d06536b49c3fee6c3d0b6ff0fb2bf13a314f57c953b8f4d93bf88e70418010200090502533b85cd021b0200520910a42704b92866382a47200419110200060502533b85cd000a091042ce2c64bc0ba99214b2009e26b26852c8b13b10c35768e40e78fbbb48bd084100a0c79d9ea0844fa5853dd3c85ff3ecae6f2c9dd6c557aa04008bbbc964cd65b9b8299d4ebf31f41cc7264b8cf33a00e82c5af022331fac79efc9563a822497ba012953cefe2629f1242fcdcb911dbb2315985bab060bfd58261ace3c654bdbbe2e8ed27a46e836490145c86dc7bae15c011f7e1ffc33730109b9338cd9f483e7cef3d2f396aab5bd80efb6646d7e778270ee99d934d187dd98"
const revokedKeyHex = "988d045331ce82010400c4fdf7b40a5477f206e6ee278eaef888ca73bf9128a9eef9f2f1ddb8b7b71a4c07cfa241f028a04edb405e4d916c61d6beabc333813dc7b484d2b3c52ee233c6a79b1eea4e9cc51596ba9cd5ac5aeb9df62d86ea051055b79d03f8a4fa9f38386f5bd17529138f3325d46801514ea9047977e0829ed728e68636802796801be10011010001889f04200102000905025331d0e3021d03000a0910a401d9f09a34f7c042aa040086631196405b7e6af71026b88e98012eab44aa9849f6ef3fa930c7c9f23deaedba9db1538830f8652fb7648ec3fcade8dbcbf9eaf428e83c6cbcc272201bfe2fbb90d41963397a7c0637a1a9d9448ce695d9790db2dc95433ad7be19eb3de72dacf1d6db82c3644c13eae2a3d072b99bb341debba012c5ce4006a7d34a1f4b94b444526567205265766f6b657220283c52656727732022424d204261726973746122204b657920262530305c303e5c29203c72656740626d626172697374612e636f2e61753e88b704130102002205025331ce82021b03060b090807030206150802090a0b0416020301021e01021780000a0910a401d9f09a34f7c0019c03f75edfbeb6a73e7225ad3cc52724e2872e04260d7daf0d693c170d8c4b243b8767bc7785763533febc62ec2600c30603c433c095453ede59ff2fcabeb84ce32e0ed9d5cf15ffcbc816202b64370d4d77c1e9077d74e94a16fb4fa2e5bec23a56d7a73cf275f91691ae1801a976fcde09e981a2f6327ac27ea1fecf3185df0d56889c04100102000605025331cfb5000a0910fe9645554e8266b64b4303fc084075396674fb6f778d302ac07cef6bc0b5d07b66b2004c44aef711cbac79617ef06d836b4957522d8772dd94bf41a2f4ac8b1ee6d70c57503f837445a74765a076d07b829b8111fc2a918423ddb817ead7ca2a613ef0bfb9c6b3562aec6c3cf3c75ef3031d81d95f6563e4cdcc9960bcb386c5d757b104fcca5fe11fc709df884604101102000605025331cfe7000a09107b15a67f0b3ddc0317f6009e360beea58f29c1d963a22b962b80788c3fa6c84e009d148cfde6b351469b8eae91187eff07ad9d08fcaab88d045331ce820104009f25e20a42b904f3fa555530fe5c46737cf7bd076c35a2a0d22b11f7e0b61a69320b768f4a80fe13980ce380d1cfc4a0cd8fbe2d2e2ef85416668b77208baa65bf973fe8e500e78cc310d7c8705cdb34328bf80e24f0385fce5845c33bc7943cf6b11b02348a23da0bf6428e57c05135f2dc6bd7c1ce325d666d5a5fd2fd5e410011010001889f04180102000905025331ce82021b0c000a0910a401d9f09a34f7c0418003fe34feafcbeaef348a800a0d908a7a6809cc7304017d820f70f0474d5e23cb17e38b67dc6dca282c6ca00961f4ec9edf2738d0f087b1d81e4871ef08e1798010863afb4eac4c44a376cb343be929c5be66a78cfd4456ae9ec6a99d97f4e1c3ff3583351db2147a65c0acef5c003fb544ab3a2e2dc4d43646f58b811a6c3a369d1f"
const revokedSubkeyHex = "988d04533121f6010400aefc803a3e4bb1a61c86e8a86d2726c6a43e0079e9f2713f1fa017e9854c83877f4aced8e331d675c67ea83ddab80aacbfa0b9040bb12d96f5a3d6be09455e2a76546cbd21677537db941cab710216b6d24ec277ee0bd65b910f416737ed120f6b93a9d3b306245c8cfd8394606fdb462e5cf43c551438d2864506c63367fc890011010001b41d416c696365203c616c69636540626d626172697374612e636f2e61753e88bb041301020025021b03060b090807030206150802090a0b0416020301021e01021780050253312798021901000a09104ef7e4beccde97f015a803ff5448437780f63263b0df8442a995e7f76c221351a51edd06f2063d8166cf3157aada4923dfc44aa0f2a6a4da5cf83b7fe722ba8ab416c976e77c6b5682e7f1069026673bd0de56ba06fd5d7a9f177607f277d9b55ff940a638c3e68525c67517e2b3d976899b93ca267f705b3e5efad7d61220e96b618a4497eab8d04403d23f8846041011020006050253312910000a09107b15a67f0b3ddc03d96e009f50b6365d86c4be5d5e9d0ea42d5e56f5794c617700a0ab274e19c2827780016d23417ce89e0a2c0d987d889c04100102000605025331cf7a000a0910a401d9f09a34f7c0ee970400aca292f213041c9f3b3fc49148cbda9d84afee6183c8dd6c5ff2600b29482db5fecd4303797be1ee6d544a20a858080fec43412061c9a71fae4039fd58013b4ae341273e6c66ad4c7cdd9e68245bedb260562e7b166f2461a1032f2b38c0e0e5715fb3d1656979e052b55ca827a76f872b78a9fdae64bc298170bfcebedc1271b41a416c696365203c616c696365407379646973702e6f722e61753e88b804130102002205025331278b021b03060b090807030206150802090a0b0416020301021e01021780000a09104ef7e4beccde97f06a7003fa03c3af68d272ebc1fa08aa72a03b02189c26496a2833d90450801c4e42c5b5f51ad96ce2d2c9cef4b7c02a6a2fcf1412d6a2d486098eb762f5010a201819c17fd2888aec8eda20c65a3b75744de7ee5cc8ac7bfc470cbe3cb982720405a27a3c6a8c229cfe36905f881b02ed5680f6a8f05866efb9d6c5844897e631deb949ca8846041011020006050253312910000a09107b15a67f0b3ddc0347bc009f7fa35db59147469eb6f2c5aaf6428accb138b22800a0caa2f5f0874bacc5909c652a57a31beda65eddd5889c04100102000605025331cf7a000a0910a401d9f09a34f7c0316403ff46f2a5c101256627f16384d34a38fb47a6c88ba60506843e532d91614339fccae5f884a5741e7582ffaf292ba38ee10a270a05f139bde3814b6a077e8cd2db0f105ebea2a83af70d385f13b507fac2ad93ff79d84950328bb86f3074745a8b7f9b64990fb142e2a12976e27e8d09a28dc5621f957ac49091116da410ac3cbde1b88d04533121f6010400cbd785b56905e4192e2fb62a720727d43c4fa487821203cf72138b884b78b701093243e1d8c92a0248a6c0203a5a88693da34af357499abacaf4b3309c640797d03093870a323b4b6f37865f6eaa2838148a67df4735d43a90ca87942554cdf1c4a751b1e75f9fd4ce4e97e278d6c1c7ed59d33441df7d084f3f02beb68896c70011010001889f0418010200090502533121f6021b0c000a09104ef7e4beccde97f0b98b03fc0a5ccf6a372995835a2f5da33b282a7d612c0ab2a97f59cf9fff73e9110981aac2858c41399afa29624a7fd8a0add11654e3d882c0fd199e161bdad65e5e2548f7b68a437ea64293db1246e3011cbb94dc1bcdeaf0f2539bd88ff16d95547144d97cead6a8c5927660a91e6db0d16eb36b7b49a3525b54d1644e65599b032b7eb901a204533127a0110400bd3edaa09eff9809c4edc2c2a0ebe52e53c50a19c1e49ab78e6167bf61473bb08f2050d78a5cbbc6ed66aff7b42cd503f16b4a0b99fa1609681fca9b7ce2bbb1a5b3864d6cdda4d7ef7849d156d534dea30fb0efb9e4cf8959a2b2ce623905882d5430b995a15c3b9fe92906086788b891002924f94abe139b42cbbfaaabe42f00a0b65dc1a1ad27d798adbcb5b5ad02d2688c89477b03ff4eebb6f7b15a73b96a96bed201c0e5e4ea27e4c6e2dd1005b94d4b90137a5b1cf5e01c6226c070c4cc999938101578877ee76d296b9aab8246d57049caacf489e80a3f40589cade790a020b1ac146d6f7a6241184b8c7fcde680eae3188f5dcbe846d7f7bdad34f6fcfca08413e19c1d5df83fc7c7c627d493492e009c2f52a80400a2fe82de87136fd2e8845888c4431b032ba29d9a29a804277e31002a8201fb8591a3e55c7a0d0881496caf8b9fb07544a5a4879291d0dc026a0ea9e5bd88eb4aa4947bbd694b25012e208a250d65ddc6f1eea59d3aed3b4ec15fcab85e2afaa23a40ab1ef9ce3e11e1bc1c34a0e758e7aa64deb8739276df0af7d4121f834a9b88e70418010200090502533127a0021b02005209104ef7e4beccde97f047200419110200060502533127a0000a0910dbce4ee19529437fe045009c0b32f5ead48ee8a7e98fac0dea3d3e6c0e2c552500a0ad71fadc5007cfaf842d9b7db3335a8cdad15d3d1a6404009b08e2c68fe8f3b45c1bb72a4b3278cdf3012aa0f229883ad74aa1f6000bb90b18301b2f85372ca5d6b9bf478d235b733b1b197d19ccca48e9daf8e890cb64546b4ce1b178faccfff07003c172a2d4f5ebaba9f57153955f3f61a9b80a4f5cb959908f8b211b03b7026a8a82fc612bfedd3794969bcf458c4ce92be215a1176ab88d045331d144010400a5063000c5aaf34953c1aa3bfc95045b3aab9882b9a8027fecfe2142dc6b47ba8aca667399990244d513dd0504716908c17d92c65e74219e004f7b83fc125e575dd58efec3ab6dd22e3580106998523dea42ec75bf9aa111734c82df54630bebdff20fe981cfc36c76f865eb1c2fb62c9e85bc3a6e5015a361a2eb1c8431578d0011010001889f04280102000905025331d433021d03000a09104ef7e4beccde97f02e5503ff5e0630d1b65291f4882b6d40a29da4616bb5088717d469fbcc3648b8276de04a04988b1f1b9f3e18f52265c1f8b6c85861691c1a6b8a3a25a1809a0b32ad330aec5667cb4262f4450649184e8113849b05e5ad06a316ea80c001e8e71838190339a6e48bbde30647bcf245134b9a97fa875c1d83a9862cae87ffd7e2c4ce3a1b89013d04180102000905025331d144021b0200a809104ef7e4beccde97f09d2004190102000605025331d144000a0910677815e371c2fd23522203fe22ab62b8e7a151383cea3edd3a12995693911426f8ccf125e1f6426388c0010f88d9ca7da2224aee8d1c12135998640c5e1813d55a93df472faae75bef858457248db41b4505827590aeccf6f9eb646da7f980655dd3050c6897feddddaca90676dee856d66db8923477d251712bb9b3186b4d0114daf7d6b59272b53218dd1da94a03ff64006fcbe71211e5daecd9961fba66cdb6de3f914882c58ba5beddeba7dcb950c1156d7fba18c19ea880dccc800eae335deec34e3b84ac75ffa24864f782f87815cda1c0f634b3dd2fa67cea30811d21723d21d9551fa12ccbcfa62b6d3a15d01307b99925707992556d50065505b090aadb8579083a20fe65bd2a270da9b011"
const missingCrossSignatureKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
Charset: UTF-8

mQENBFMYynYBCACVOZ3/e8Bm2b9KH9QyIlHGo/i1bnkpqsgXj8tpJ2MIUOnXMMAY
ztW7kKFLCmgVdLIC0vSoLA4yhaLcMojznh/2CcUglZeb6Ao8Gtelr//Rd5DRfPpG
zqcfUo+m+eO1co2Orabw0tZDfGpg5p3AYl0hmxhUyYSc/xUq93xL1UJzBFgYXY54
QsM8dgeQgFseSk/YvdP5SMx1ev+eraUyiiUtWzWrWC1TdyRa5p4UZg6Rkoppf+WJ
QrW6BWrhAtqATHc8ozV7uJjeONjUEq24roRc/OFZdmQQGK6yrzKnnbA6MdHhqpdo
9kWDcXYb7pSE63Lc+OBa5X2GUVvXJLS/3nrtABEBAAG0F2ludmFsaWQtc2lnbmlu
Zy1zdWJrZXlziQEoBBMBAgASBQJTnKB5AhsBAgsHAhUIAh4BAAoJEO3UDQUIHpI/
dN4H/idX4FQ1LIZCnpHS/oxoWQWfpRgdKAEM0qCqjMgiipJeEwSQbqjTCynuh5/R
JlODDz85ABR06aoF4l5ebGLQWFCYifPnJZ/Yf5OYcMGtb7dIbqxWVFL9iLMO/oDL
ioI3dotjPui5e+2hI9pVH1UHB/bZ/GvMGo6Zg0XxLPolKQODMVjpjLAQ0YJ3spew
RAmOGre6tIvbDsMBnm8qREt7a07cBJ6XK7xjxYaZHQBiHVxyEWDa6gyANONx8duW
/fhQ/zDTnyVM/ik6VO0Ty9BhPpcEYLFwh5c1ilFari1ta3e6qKo6ZGa9YMk/REhu
yBHd9nTkI+0CiQUmbckUiVjDKKe5AQ0EUxjKdgEIAJcXQeP+NmuciE99YcJoffxv
2gVLU4ZXBNHEaP0mgaJ1+tmMD089vUQAcyGRvw8jfsNsVZQIOAuRxY94aHQhIRHR
bUzBN28ofo/AJJtfx62C15xt6fDKRV6HXYqAiygrHIpEoRLyiN69iScUsjIJeyFL
C8wa72e8pSL6dkHoaV1N9ZH/xmrJ+k0vsgkQaAh9CzYufncDxcwkoP+aOlGtX1gP
WwWoIbz0JwLEMPHBWvDDXQcQPQTYQyj+LGC9U6f9VZHN25E94subM1MjuT9OhN9Y
MLfWaaIc5WyhLFyQKW2Upofn9wSFi8ubyBnv640Dfd0rVmaWv7LNTZpoZ/GbJAMA
EQEAAYkBHwQYAQIACQUCU5ygeQIbAgAKCRDt1A0FCB6SP0zCB/sEzaVR38vpx+OQ
MMynCBJrakiqDmUZv9xtplY7zsHSQjpd6xGflbU2n+iX99Q+nav0ETQZifNUEd4N
1ljDGQejcTyKD6Pkg6wBL3x9/RJye7Zszazm4+toJXZ8xJ3800+BtaPoI39akYJm
+ijzbskvN0v/j5GOFJwQO0pPRAFtdHqRs9Kf4YanxhedB4dIUblzlIJuKsxFit6N
lgGRblagG3Vv2eBszbxzPbJjHCgVLR3RmrVezKOsZjr/2i7X+xLWIR0uD3IN1qOW
CXQxLBizEEmSNVNxsp7KPGTLnqO3bPtqFirxS9PJLIMPTPLNBY7ZYuPNTMqVIUWF
4artDmrG
=7FfJ
-----END PGP PUBLIC KEY BLOCK-----`

const invalidCrossSignatureKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBFMYynYBCACVOZ3/e8Bm2b9KH9QyIlHGo/i1bnkpqsgXj8tpJ2MIUOnXMMAY
ztW7kKFLCmgVdLIC0vSoLA4yhaLcMojznh/2CcUglZeb6Ao8Gtelr//Rd5DRfPpG
zqcfUo+m+eO1co2Orabw0tZDfGpg5p3AYl0hmxhUyYSc/xUq93xL1UJzBFgYXY54
QsM8dgeQgFseSk/YvdP5SMx1ev+eraUyiiUtWzWrWC1TdyRa5p4UZg6Rkoppf+WJ
QrW6BWrhAtqATHc8ozV7uJjeONjUEq24roRc/OFZdmQQGK6yrzKnnbA6MdHhqpdo
9kWDcXYb7pSE63Lc+OBa5X2GUVvXJLS/3nrtABEBAAG0F2ludmFsaWQtc2lnbmlu
Zy1zdWJrZXlziQEoBBMBAgASBQJTnKB5AhsBAgsHAhUIAh4BAAoJEO3UDQUIHpI/
dN4H/idX4FQ1LIZCnpHS/oxoWQWfpRgdKAEM0qCqjMgiipJeEwSQbqjTCynuh5/R
JlODDz85ABR06aoF4l5ebGLQWFCYifPnJZ/Yf5OYcMGtb7dIbqxWVFL9iLMO/oDL
ioI3dotjPui5e+2hI9pVH1UHB/bZ/GvMGo6Zg0XxLPolKQODMVjpjLAQ0YJ3spew
RAmOGre6tIvbDsMBnm8qREt7a07cBJ6XK7xjxYaZHQBiHVxyEWDa6gyANONx8duW
/fhQ/zDTnyVM/ik6VO0Ty9BhPpcEYLFwh5c1ilFari1ta3e6qKo6ZGa9YMk/REhu
yBHd9nTkI+0CiQUmbckUiVjDKKe5AQ0EUxjKdgEIAIINDqlj7X6jYKc6DjwrOkjQ
UIRWbQQar0LwmNilehmt70g5DCL1SYm9q4LcgJJ2Nhxj0/5qqsYib50OSWMcKeEe
iRXpXzv1ObpcQtI5ithp0gR53YPXBib80t3bUzomQ5UyZqAAHzMp3BKC54/vUrSK
FeRaxDzNLrCeyI00+LHNUtwghAqHvdNcsIf8VRumK8oTm3RmDh0TyjASWYbrt9c8
R1Um3zuoACOVy+mEIgIzsfHq0u7dwYwJB5+KeM7ZLx+HGIYdUYzHuUE1sLwVoELh
+SHIGHI1HDicOjzqgajShuIjj5hZTyQySVprrsLKiXS6NEwHAP20+XjayJ/R3tEA
EQEAAYkCPgQYAQIBKAUCU5ygeQIbAsBdIAQZAQIABgUCU5ygeQAKCRCpVlnFZmhO
52RJB/9uD1MSa0wjY6tHOIgquZcP3bHBvHmrHNMw9HR2wRCMO91ZkhrpdS3ZHtgb
u3/55etj0FdvDo1tb8P8FGSVtO5Vcwf5APM8sbbqoi8L951Q3i7qt847lfhu6sMl
w0LWFvPTOLHrliZHItPRjOltS1WAWfr2jUYhsU9ytaDAJmvf9DujxEOsN5G1YJep
54JCKVCkM/y585Zcnn+yxk/XwqoNQ0/iJUT9qRrZWvoeasxhl1PQcwihCwss44A+
YXaAt3hbk+6LEQuZoYS73yR3WHj+42tfm7YxRGeubXfgCEz/brETEWXMh4pe0vCL
bfWrmfSPq2rDegYcAybxRQz0lF8PAAoJEO3UDQUIHpI/exkH/0vQfdHA8g/N4T6E
i6b1CUVBAkvtdJpCATZjWPhXmShOw62gkDw306vHPilL4SCvEEi4KzG72zkp6VsB
DSRcpxCwT4mHue+duiy53/aRMtSJ+vDfiV1Vhq+3sWAck/yUtfDU9/u4eFaiNok1
8/Gd7reyuZt5CiJnpdPpjCwelK21l2w7sHAnJF55ITXdOxI8oG3BRKufz0z5lyDY
s2tXYmhhQIggdgelN8LbcMhWs/PBbtUr6uZlNJG2lW1yscD4aI529VjwJlCeo745
U7pO4eF05VViUJ2mmfoivL3tkhoTUWhx8xs8xCUcCg8DoEoSIhxtOmoTPR22Z9BL
6LCg2mg=
=Dhm4
-----END PGP PUBLIC KEY BLOCK-----`

const goodCrossSignatureKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v1

mI0EVUqeVwEEAMufHRrMPWK3gyvi0O0tABCs/oON9zV9KDZlr1a1M91ShCSFwCPo
7r80PxdWVWcj0V5h50/CJYtpN3eE/mUIgW2z1uDYQF1OzrQ8ubrksfsJvpAhENom
lTQEppv9mV8qhcM278teb7TX0pgrUHLYF5CfPdp1L957JLLXoQR/lwLVABEBAAG0
E2dvb2Qtc2lnbmluZy1zdWJrZXmIuAQTAQIAIgUCVUqeVwIbAwYLCQgHAwIGFQgC
CQoLBBYCAwECHgECF4AACgkQNRjL95IRWP69XQQAlH6+eyXJN4DZTLX78KGjHrsw
6FCvxxClEPtPUjcJy/1KCRQmtLAt9PbbA78dvgzjDeZMZqRAwdjyJhjyg/fkU2OH
7wq4ktjUu+dLcOBb+BFMEY+YjKZhf6EJuVfxoTVr5f82XNPbYHfTho9/OABKH6kv
X70PaKZhbwnwij8Nts65AaIEVUqftREEAJ3WxZfqAX0bTDbQPf2CMT2IVMGDfhK7
GyubOZgDFFjwUJQvHNvsrbeGLZ0xOBumLINyPO1amIfTgJNm1iiWFWfmnHReGcDl
y5mpYG60Mb79Whdcer7CMm3AqYh/dW4g6IB02NwZMKoUHo3PXmFLxMKXnWyJ0clw
R0LI/Qn509yXAKDh1SO20rqrBM+EAP2c5bfI98kyNwQAi3buu94qo3RR1ZbvfxgW
CKXDVm6N99jdZGNK7FbRifXqzJJDLcXZKLnstnC4Sd3uyfyf1uFhmDLIQRryn5m+
LBYHfDBPN3kdm7bsZDDq9GbTHiFZUfm/tChVKXWxkhpAmHhU/tH6GGzNSMXuIWSO
aOz3Rqq0ED4NXyNKjdF9MiwD/i83S0ZBc0LmJYt4Z10jtH2B6tYdqnAK29uQaadx
yZCX2scE09UIm32/w7pV77CKr1Cp/4OzAXS1tmFzQ+bX7DR+Gl8t4wxr57VeEMvl
BGw4Vjh3X8//m3xynxycQU18Q1zJ6PkiMyPw2owZ/nss3hpSRKFJsxMLhW3fKmKr
Ey2KiOcEGAECAAkFAlVKn7UCGwIAUgkQNRjL95IRWP5HIAQZEQIABgUCVUqftQAK
CRD98VjDN10SqkWrAKDTpEY8D8HC02E/KVC5YUI01B30wgCgurpILm20kXEDCeHp
C5pygfXw1DJrhAP+NyPJ4um/bU1I+rXaHHJYroYJs8YSweiNcwiHDQn0Engh/mVZ
SqLHvbKh2dL/RXymC3+rjPvQf5cup9bPxNMa6WagdYBNAfzWGtkVISeaQW+cTEp/
MtgVijRGXR/lGLGETPg2X3Afwn9N9bLMBkBprKgbBqU7lpaoPupxT61bL70=
=vtbN
-----END PGP PUBLIC KEY BLOCK-----`
