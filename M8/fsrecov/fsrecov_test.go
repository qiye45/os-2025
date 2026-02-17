package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const result = `hash      size    filename
d60e7d3d2b47d19418af5b0ba52406b86ec6ef83  489270  0M15CwG1yP32UPCp.bmp
1ab8c4f2e61903ae2a00d0820ea0111fac04d9d3  377394  1yh0sw8n6.bmp
1681e23d7b8bb0b36c399c065514bc04badfde79  617814  2Kbg82NaSqPga.bmp
aabd1ef8a2371dd64fb64fc7f10a0a31047d1023  902934  2pxHTrpI.bmp
3f4cfd6c2b7b788062283db37a8f06a4a4a210e4  349214  335qZ0PhcpRTxMb.bmp
77b6d70b6e52d6613b7c95e059cbfc429b3de7e0  1007862  35OZL3hvJnEf.bmp
03ec749d01bae86200189b546839fab1630766b7  649366  3DhTVVP9avTrH.bmp
d46c5781477c8b7e18e71507f2713fc5a11e4af6  590686  4QDw0lcDAIhO.bmp
bf06738214223ce3fb4651edeedc7488d28fc58d  911094  5VelGxd7.bmp
e02a7e2628fe0edfb0fc87f6725fb5b0935aa237  1203094  5YpvCYAOItJaxUBL.bmp
a4e48d5bd7be56f441f1cd77cff473f4709fb37c  355134  5wISaRF1C3r4JgP.bmp
94ff9fd78a105eda524e0ecc3e3c7e79088ba902  907970  6lHVixCC6.bmp
e55615c8bb6a5ba187771f6462e8c9a65939a8bc  332814  7biIM7Q8P9bq.bmp
e1dd4448134c8fbf66626dbd714c9e19ec90fefa  408550  8G5szK2BVe.bmp
dfc6272ba28ffd86bb288813afe67200e3da9ebb  357558  8WkOrZWB2ukhnnd5M.bmp
a776e14ab2ad3a65f166683c02735050a9e00f27  1042938  8uGiCZmSVfvwh0SZ.bmp
2f2abc7151e7ef34d07395ff9fafbe78cbb8596c  472022  9my4xesCy2E9jAN.bmp
9d7ef1ad6532e9b4ae86ef329948d0beb6491fae  1043774  ASGbmfyufH4dN0.bmp
d362e8966a9f4e6b2f057d1e2c3c454153f2df47  311054  B8siuWRm7u7gQDr.bmp
2efb8417d20275b3e7ddf3f229086958e7dd91ed  1000566  BE4gsl2y.bmp
a64be9c9e3e28542903518d192e8f97d4d8a777b  512710  BOHQPYEfWAzaSy.bmp
fb1de8d4cfa9a823868a622034279d2efb8b1d23  1420278  BSMby3blPP2g.bmp
a9d9d319d3ca10870e31d1595f00d5c27dee7f1b  932742  C0NrniO8Tu5T.bmp
29a3da15fd10cb619a3fdf12520f5c9ec7075619  641214  C6bsubODgR3P.bmp
5afaa51e0f0aef45d9a2482108e4f418cf9337fb  725658  CJuVzZpj9AXIQE.bmp
5879e4ddd92f347577bdbcc49adf5adc5c186d24  1346454  CfWT9haBXoTJ.bmp
3d9e4babbe86de760528da349c51062d1c63c30a  308934  ChUOLUWrtaNmMeIdG.bmp
cf3542551edba0a29d8fafd1aa5d0002910b029e  310514  Dnf6EC9Wn.bmp
a40e45f3f1a25a55abc916f7fc7f8a85d73f830b  350814  EhDqKXHi3AUn.bmp
5e84aa3e615f95024d2a55f543cdceab63a30c44  1037286  FaccLox6fC9h.bmp
84c34532c462f5ca7222f2d738105b2c426760eb  301258  GVTL53R3j6ha.bmp
80270bb73ad0331cc2889604350c9f9a192de079  330730  GolAJQfSet.bmp
cabf852574ecca866278329de8a1d1a82189165c  1185018  Hj9Afs61.bmp
38fe0c7bfe9ed7c931843dcf0ea2b29688297a75  410934  IEyXmSDCCO4Q.bmp
c5170b33cc819ffc19bc0e7fc9ef9e7bbc00a514  509814  K3P0frhdzAz.bmp
25f90ec06c1e2cedc87a9c838c24c34f4ba31fa8  1051518  KRFgoEA52w.bmp
4d6b2b2bcbc10e04ad64350a61abe77effb79a42  309942  Le4I4eLH.bmp
53962ea913ee0da56141aa2d2752cbd3d3973771  1031446  Lh3gk68UqXC.bmp
5274d124d86be0cb83f9a2bc6f19fe7d1dc0aef6  473814  NUiFpilAZL1w.bmp
befc711f13af3adacd38a76579b500a3580d79ed  957766  NkeMerFkx02.bmp
d93a6970969741e581dee787b0b0d27bd68cd97e  426438  O6GneCnOfJ7.bmp
894de00f8f24df0be6fc28a153b06db7ad35d7d1  305638  OqpvBJ3LdOJ7.bmp
f7709bf0e29f85d5eadd34bf2e4d342cad067637  433054  P4LECOKK.bmp
7c75729e797338a5b1baf2cb0c41eaa65605e74d  556854  PNTuejwC2.bmp
2663c82d2de3c6a40dd544817d11f4922470389e  496674  PxyLldumMJ7x9bhVp.bmp
a1a958308141554f7dc0aafc32f79a83549c28c7  614758  QEh5F9z7e.bmp
2ea146afa7f84d696a3d29f48423658164544833  339774  REPlnIEiLXRund.bmp
537c8d34341af415313d67efe8fa591dc7910ae4  1355286  S0A5Dyr2uLL.bmp
9ae2b885673a9ec2cd4f0cb19ea3ff6fe0ed77fa  408098  TAYFI98zRuFHE2.bmp
b4d86bde39a8704b8db8f17003b45f0789a155d4  368058  Tn23kpRhjIUK1Tr.bmp
2cf9ec47bb6e7f4f78cffb33247f048837c5e300  721978  UJVORXkzJG.bmp
b1c3493f6ad60f5e6942539b0559205b903af4c4  564822  Utq6k0l5A4nuN0q.bmp
97e9f140bc7aff5525f1ac5ce0bb7c75c9958021  395382  Vc5voqD38uOuc.bmp
98dee77fccdda83f577f255122303758565af1e2  480222  WDESkd1ohYoeScb0.bmp
c5e72335ba4cb2ac5d2b581a12de2d4ceb7a5af8  1077838  WkoT3Hhve6K4a5.bmp
3c37d7c2a9f56bdcb809b054bcb5d668fa67ce3b  469530  YpudsKsAkEzDBoJ.bmp
3c324066045f9b1cc6b0950d4652b205587b7e64  686854  aAIzYjw4CD.bmp
3424b6253defdca5ab2c816cfb1f2bbdb3ca76ec  1269054  aCYph3JKNaQ5.bmp
49af0c1aad982744d2ebb2e67eaeaf8332ff8732  333054  aMm0Fgjgwdde.bmp
b33ec9a67b7304dc9b3eda8bd31726b83a851a54  462734  aUqJhfhLguLzI56.bmp
8070af25a450771da1eae9d2126396ed6778778a  991770  bPtMCWiwCuGKe8.bmp
d4806efff96df9dc1ae3c25c79838b5c9ea90ab0  782262  bhxRNph5PX.bmp
86f758470f2be0164ff71a024dae4db95f7956ed  319230  cLFhqxWoVDu7d4AF.bmp
f9b346a8df95d82e576a30228bb2aba2396c3f71  869570  cwPgyDeTuhhf.bmp
1538b092d4961cea7ef6c190bfb432847ac47310  661950  dwuwDLbM.bmp
cf60d5c1965db724854b9fdd4deadfc6d25870a3  476614  e8XGTmRSoGBp.bmp
8940595fb105e5825e8a43add015ff26cb2957db  854394  eDMwaMHsNO2TVn.bmp
6e4afd6d293d0b8e969442fe5f5583febb3fb1f9  303162  emvzYdDWfu9i28IGJ.bmp
cafd0cde89647fe4c2457f3f0a4cba865c215a35  1274334  izV3ngDlqtNQvI359.bmp
9b694c640b71edb91e9f1322c03617c2bfdcc511  729294  j1KVXozUClC.bmp
6eec4f7f578056e8148015e8dcb5270672c47210  376094  jGptw2hnT.bmp
04bbc0049f28caa430b26192073c348b1a99a076  868278  jJQ0E9ujM6.bmp
2cea6f5ea98b2d487e3cc32a3308597dfe7da7c8  792102  jxtrT1VRRrr.bmp
1f23b0c13aea51b8a3c388e67e06e5410e504ecf  337558  k43VQUbabT0tC54Y.bmp
3a768b40bf66cf8afd20ac93b903973b21e031d0  997430  lgmdA1fW9NZu4T.bmp
d3ae99abb401b06cac5fc3f149f0c373cd0af053  601254  ljaGhx0jOu5r7ge.bmp
50097799b816c48f1a7c1a88cd285025b85172c0  1038270  lulFqviqr.bmp
5108c6653024f398d564f30dc674cb286f29d0da  806150  mwaYW7RAfDfDPKY0.bmp
663699b660831db55071c0b0ff320ca35bda23a3  576086  p6FaDZ3Z8GxJaV2L.bmp
9b6931a8126156ccaf40e263b4ba423a4ad4193e  1029690  pEmFLlmuB.bmp
8b5cc9b357e3e3b556063f80d2192ad7c1417f71  500022  ptWAWyKPZrvz.bmp
625b7508940cb3234363fa9308b2bc7ca0a78518  403014  q43vVNgf6SLsZ9lB0.bmp
0c08b4b1ef563d9e35fbc3b5bc6a813eb630078a  455442  qD9yubK8hHkYUURK.bmp
7cd8bf73ec91b56745ec5c12b8ca8cec05fdfbbc  1085242  rVVtFPtJ.bmp
3ad258494666771200c62bc8a60588ae32cfe854  819078  rgwfuyGZAfPrLw6n.bmp
f7c74bcb0e0701bf3cc4d15328f8e3c10709cefb  529014  rnS6rbKwG6R2h4krD.bmp
2f573bdbc9609f5845a63a64b28b00443475a680  1164270  tEzVaAJO1.bmp
895482516ee54fee57f03d38c41dacafb8422471  1292250  u6wlQF01C451.bmp
f9d8cebbabbe0fcdb99061a5c5466ad0b326acc2  907822  u7TOeSELhp.bmp
68278b919a2cc1d1253e3ec354794b5d21425203  968054  uf9jm62NlUHYMknj.bmp
548466f58810eae6fce81a283cadcdbbed8ea6b1  359606  uqPcdWI64PpuXGYsM.bmp
f1def12736c2e294582af8885f41384623c818f3  553554  vXyZkWFGf486oGM.bmp
5f210448e64abdd5d87b01fb3e848a658b209459  566026  vdLqNhwlgd5.bmp
ae8e33136c7fef924131ef16f83d6d10beaeae9f  611274  wCxJzyUXtXjCWnpU.bmp
be8989c7385a43ea421e1d2ea98b47822772dec0  852094  whCeEdLoscu89d5Km.bmp
e8954cefdd31314ad94d371ac9b6b8caa0a3ce17  1036078  zOU0Xf3NSQyiJESqD.bmp
3de7d1e949ae7f260119ac40a4560cab6885c525  426006  zXcUJJDdm.bmp
`

type ExpectedFile struct {
	SHA1 string
	Size int64
}

// TestRecoveryAccuracy 测试文件恢复准确率
func TestRecoveryAccuracy(t *testing.T) {
	imageFile := "fsrecov.img"

	// 检查镜像文件是否存在
	if _, err := os.Stat(imageFile); os.IsNotExist(err) {
		t.Skipf("Test image %s not found, skipping accuracy test", imageFile)
	}

	// 解析预期结果
	expected := parseExpectedResults(result)
	if len(expected) == 0 {
		t.Fatal("No expected results to compare against")
	}

	// 运行恢复程序
	data, header, err := mapDisk(imageFile)
	if err != nil {
		t.Fatalf("Failed to map disk: %v", err)
	}

	// 扫描簇并解析目录项
	clusters := scanClusters(data, header)
	files := parseDirectoryEntries(clusters)

	// 恢复文件并计算SHA1
	type RecoveredFile struct {
		SHA1 string
		Size int64
	}
	recovered := make(map[string]RecoveredFile) // filename -> {sha1, size}
	for _, file := range files {
		bmpData, err := recoverBMPFile(file, clusters, header, data)
		if err != nil {
			continue
		}
		sha1sum := calculateSHA1(bmpData)
		recovered[file.Name] = RecoveredFile{
			SHA1: sha1sum,
			Size: file.Size,
		}
	}

	// 计算准确率
	correctFiles := 0
	correctSizes := 0
	totalExpected := len(expected)

	for filename, expectedFile := range expected {
		if recoveredFile, ok := recovered[filename]; ok {
			if recoveredFile.SHA1 == expectedFile.SHA1 {
				correctFiles++
			}
			if recoveredFile.Size == expectedFile.Size {
				correctSizes++
			}
		}
	}

	filenameAccuracy := float64(correctFiles) / float64(totalExpected) * 100
	sizeAccuracy := float64(correctSizes) / float64(totalExpected) * 100

	t.Logf("Total expected files: %d", totalExpected)
	t.Logf("Correctly recovered files (SHA1 match): %d", correctFiles)
	t.Logf("Correctly recovered sizes: %d", correctSizes)
	t.Logf("Filename accuracy: %.2f%%", filenameAccuracy)
	t.Logf("File size accuracy: %.2f%%", sizeAccuracy)

	// 根据要求的准确率标准进行判断
	if filenameAccuracy < 10 {
		t.Errorf("Filename accuracy %.2f%% is below 10%% (easy test case threshold)", filenameAccuracy)
	} else if filenameAccuracy >= 10 && filenameAccuracy < 50 {
		t.Logf("✓ Passed easy test cases (>10%% filename accuracy)")
	} else if filenameAccuracy >= 50 && filenameAccuracy < 75 {
		t.Logf("✓ Passed easy test cases and one hard test case (>50%% filename accuracy)")
	} else if filenameAccuracy >= 75 {
		t.Logf("✓ High filename accuracy (>75%%)")
	}

	// 文件大小准确率判断
	if sizeAccuracy < 10 {
		t.Errorf("File size accuracy %.2f%% is below 10%%", sizeAccuracy)
	} else if sizeAccuracy >= 10 && sizeAccuracy < 50 {
		t.Logf("✓ Basic file size accuracy (>10%%)")
	} else if sizeAccuracy >= 50 && sizeAccuracy < 75 {
		t.Logf("✓ Good file size accuracy (>50%%)")
	} else if sizeAccuracy >= 75 {
		t.Logf("✓ High file size accuracy (>75%%)")
	}
}

// parseExpectedResults 解析预期结果字符串
func parseExpectedResults(resultStr string) map[string]ExpectedFile {
	results := make(map[string]ExpectedFile)
	lines := strings.Split(resultStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "hash") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			sha1sum := fields[0]
			var size int64
			_, err := fmt.Sscanf(fields[1], "%d", &size)
			if err != nil {
				continue
			}
			filename := fields[2]
			results[filename] = ExpectedFile{
				SHA1: sha1sum,
				Size: size,
			}
		}
	}

	return results
}
