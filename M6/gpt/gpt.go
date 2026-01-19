package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
)

// ParameterTensors 模型参数
type ParameterTensors struct {
	wte      []float32 // (V, C)
	wpe      []float32 // (maxT, C)
	ln1w     []float32 // (L, C)
	ln1b     []float32 // (L, C)
	qkvw     []float32 // (L, 3*C, C)
	qkvb     []float32 // (L, 3*C)
	attprojw []float32 // (L, C, C)
	attprojb []float32 // (L, C)
	ln2w     []float32 // (L, C)
	ln2b     []float32 // (L, C)
	fcw      []float32 // (L, 4*C, C)
	fcb      []float32 // (L, 4*C)
	fcprojw  []float32 // (L, C, 4*C)
	fcprojb  []float32 // (L, C)
	lnfw     []float32 // (C)
	lnfb     []float32 // (C)
}

// ActivationTensors 激活值
type ActivationTensors struct {
	encoded   []float32 // (B, T, C)
	ln1       []float32 // (L, B, T, C)
	ln1Mean   []float32 // (L, B, T)
	ln1Rstd   []float32 // (L, B, T)
	qkv       []float32 // (L, B, T, 3*C)
	atty      []float32 // (L, B, T, C)
	preatt    []float32 // (L, B, NH, T, T)
	att       []float32 // (L, B, NH, T, T)
	attproj   []float32 // (L, B, T, C)
	residual2 []float32 // (L, B, T, C)
	ln2       []float32 // (L, B, T, C)
	ln2Mean   []float32 // (L, B, T)
	ln2Rstd   []float32 // (L, B, T)
	fch       []float32 // (L, B, T, 4*C)
	fchGelu   []float32 // (L, B, T, 4*C)
	fcproj    []float32 // (L, B, T, C)
	residual3 []float32 // (L, B, T, C)
	lnf       []float32 // (B, T, C)
	lnfMean   []float32 // (B, T)
	lnfRstd   []float32 // (B, T)
	logits    []float32 // (B, T, V)
	probs     []float32 // (B, T, V)
	losses    []float32 // (B, T)
}

// GPT2Config 配置
type GPT2Config struct {
	maxSeqLen int
	vocabSize int
	numLayers int
	numHeads  int
	channels  int
}

// GPT2 模型
type GPT2 struct {
	config    GPT2Config
	params    ParameterTensors
	acts      ActivationTensors
	batchSize int
	seqLen    int
}

func encoderForward(out []float32, inp []int, wte, wpe []float32, B, T, C int) {
	// out is (B,T,C). At each position (b,t), a C-dimensional vector summarizing token & position
	// inp is (B,T) of integers, holding the token ids at each (b,t) position
	// wte is (V,C) of token embeddings
	// wpe is (maxT,C) of position embeddings
	for b := 0; b < B; b++ {
		for t := 0; t < T; t++ {
			// seek to the output position in out[b,t,:]
			outBt := out[b*T*C+t*C:]
			// get the index of the token at inp[b, t]
			ix := inp[b*T+t]
			// seek to the position in wte corresponding to the token
			wteIx := wte[ix*C:]
			// seek to the position in wpe corresponding to the position
			wpeT := wpe[t*C:]
			// add the two vectors and store the result in out[b,t,:]
			for i := 0; i < C; i++ {
				outBt[i] = wteIx[i] + wpeT[i]
			}
		}
	}
}

func layernormForward(out, mean, rstd, inp, weight, bias []float32, B, T, C int) {
	// both inp and out are (B,T,C) of the activations
	// mean and rstd are (B,T) buffers
	eps := float32(1e-5)
	for b := 0; b < B; b++ {
		for t := 0; t < T; t++ {
			// seek to the input position inp[b,t,:]
			x := inp[b*T*C+t*C:]
			// calculate the mean
			m := float32(0.0)
			for i := 0; i < C; i++ {
				m += x[i]
			}
			m = m / float32(C)
			// calculate the variance (without any bias correction)
			v := float32(0.0)
			for i := 0; i < C; i++ {
				xshift := x[i] - m
				v += xshift * xshift
			}
			v = v / float32(C)
			// calculate the rstd (reciprocal standard deviation)
			s := float32(1.0) / float32(math.Sqrt(float64(v+eps)))
			// seek to the output position in out[b,t,:]
			outBt := out[b*T*C+t*C:]
			for i := 0; i < C; i++ {
				n := s * (x[i] - m)        // normalize
				o := n*weight[i] + bias[i] // scale and shift
				outBt[i] = o               // write
			}
			// cache the mean and rstd for the backward pass later
			mean[b*T+t] = m
			rstd[b*T+t] = s
		}
	}
}

func matmulForward(out, inp, weight, bias []float32, B, T, C, OC int) {
	// inp is (B,T,C), weight is (OC, C), bias is (OC)
	// out will be (B,T,OC)
	for b := 0; b < B; b++ {
		for t := 0; t < T; t++ {
			outBt := out[b*T*OC+t*OC:]
			inpBt := inp[b*T*C+t*C:]
			for o := 0; o < OC; o++ {
				val := float32(0.0)
				if bias != nil {
					val = bias[o]
				}
				wrow := weight[o*C:]
				for i := 0; i < C; i++ {
					val += inpBt[i] * wrow[i]
				}
				outBt[o] = val
			}
		}
	}
}

func attentionForward(out, preatt, att, inp []float32, B, T, C, NH int) {
	// input is (B, T, 3C) holding the query, key, value (Q, K, V) vectors
	// preatt, att are (B, NH, T, T)
	// output is (B, T, C)
	C3 := C * 3
	hs := C / NH // head size
	scale := float32(1.0 / math.Sqrt(float64(hs)))

	for b := 0; b < B; b++ {
		for t := 0; t < T; t++ {
			for h := 0; h < NH; h++ {
				queryT := inp[b*T*C3+t*C3+h*hs:]
				preattBth := preatt[b*NH*T*T+h*T*T+t*T:]
				attBth := att[b*NH*T*T+h*T*T+t*T:]

				// pass 1: calculate query dot key and maxval
				maxval := float32(-10000.0)
				for t2 := 0; t2 <= t; t2++ {
					keyT2 := inp[b*T*C3+t2*C3+h*hs+C:] // +C because it's key

					// (query_t) dot (key_t2)
					val := float32(0.0)
					for i := 0; i < hs; i++ {
						val += queryT[i] * keyT2[i]
					}
					val *= scale
					if val > maxval {
						maxval = val
					}

					preattBth[t2] = val
				}

				// pass 2: calculate the exp and keep track of sum
				expsum := float32(0.0)
				for t2 := 0; t2 <= t; t2++ {
					expv := float32(math.Exp(float64(preattBth[t2] - maxval)))
					expsum += expv
					attBth[t2] = expv
				}
				expsumInv := float32(0.0)
				if expsum != 0.0 {
					expsumInv = 1.0 / expsum
				}

				// pass 3: normalize to get the softmax
				for t2 := 0; t2 < T; t2++ {
					if t2 <= t {
						attBth[t2] *= expsumInv
					} else {
						attBth[t2] = 0.0
					}
				}

				// pass 4: accumulate weighted values into the output of attention
				outBth := out[b*T*C+t*C+h*hs:]
				for i := 0; i < hs; i++ {
					outBth[i] = 0.0
				}
				for t2 := 0; t2 <= t; t2++ {
					valueT2 := inp[b*T*C3+t2*C3+h*hs+C*2:] // +C*2 because it's value
					attBtht2 := attBth[t2]
					for i := 0; i < hs; i++ {
						outBth[i] += attBtht2 * valueT2[i]
					}
				}
			}
		}
	}
}

const GELU_SCALING_FACTOR = float32(0.7978845608) // sqrt(2.0 / M_PI)

func geluForward(out, inp []float32, N int) {
	// (approximate) GeLU elementwise non-linearity
	for i := 0; i < N; i++ {
		x := inp[i]
		cube := 0.044715 * x * x * x
		out[i] = 0.5 * x * (1.0 + float32(math.Tanh(float64(GELU_SCALING_FACTOR*(x+cube)))))
	}
}

func residualForward(out, inp1, inp2 []float32, N int) {
	for i := 0; i < N; i++ {
		out[i] = inp1[i] + inp2[i]
	}
}

func softmaxForward(probs, logits []float32, B, T, V int) {
	// output: probs are (B,T,V) of the probabilities
	// input: logits is (B,T,V) of the unnormalized log probabilities
	for b := 0; b < B; b++ {
		for t := 0; t < T; t++ {
			// probs <- softmax(logits)
			logitsBt := logits[b*T*V+t*V:]
			probsBt := probs[b*T*V+t*V:]

			// maxval is only calculated and subtracted for numerical stability
			maxval := float32(-10000.0)
			for i := 0; i < V; i++ {
				if logitsBt[i] > maxval {
					maxval = logitsBt[i]
				}
			}
			sum := float32(0.0)
			for i := 0; i < V; i++ {
				probsBt[i] = float32(math.Exp(float64(logitsBt[i] - maxval)))
				sum += probsBt[i]
			}
			for i := 0; i < V; i++ {
				probsBt[i] /= sum
			}
		}
	}
}

func gpt2BuildFromCheckpoint(model *GPT2, checkpointPath string) error {
	file, err := os.Open(checkpointPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取header
	var header [256]int32
	if err := binary.Read(file, binary.LittleEndian, &header); err != nil {
		return err
	}

	if header[0] != 20240326 {
		return fmt.Errorf("bad magic model file")
	}

	// 读取超参数
	model.config.maxSeqLen = int(header[2])
	model.config.vocabSize = int(header[3])
	model.config.numLayers = int(header[4])
	model.config.numHeads = int(header[5])
	model.config.channels = int(header[6])

	maxT := model.config.maxSeqLen
	V := model.config.vocabSize
	L := model.config.numLayers
	C := model.config.channels

	// 计算参数大小
	paramSizes := []int{
		V * C,           // wte
		maxT * C,        // wpe
		L * C,           // ln1w
		L * C,           // ln1b
		L * (3 * C) * C, // qkvw
		L * (3 * C),     // qkvb
		L * C * C,       // attprojw
		L * C,           // attprojb
		L * C,           // ln2w
		L * C,           // ln2b
		L * (4 * C) * C, // fcw
		L * (4 * C),     // fcb
		L * C * (4 * C), // fcprojw
		L * C,           // fcprojb
		C,               // lnfw
		C,               // lnfb
	}

	// 计算总参数数量
	numParameters := 0
	for _, size := range paramSizes {
		numParameters += size
	}

	// 分配内存并读取参数
	paramsMemory := make([]float32, numParameters)
	if err := binary.Read(file, binary.LittleEndian, paramsMemory); err != nil {
		return err
	}

	// 将参数指向正确的位置
	offset := 0
	model.params.wte = paramsMemory[offset : offset+paramSizes[0]]
	offset += paramSizes[0]
	model.params.wpe = paramsMemory[offset : offset+paramSizes[1]]
	offset += paramSizes[1]
	model.params.ln1w = paramsMemory[offset : offset+paramSizes[2]]
	offset += paramSizes[2]
	model.params.ln1b = paramsMemory[offset : offset+paramSizes[3]]
	offset += paramSizes[3]
	model.params.qkvw = paramsMemory[offset : offset+paramSizes[4]]
	offset += paramSizes[4]
	model.params.qkvb = paramsMemory[offset : offset+paramSizes[5]]
	offset += paramSizes[5]
	model.params.attprojw = paramsMemory[offset : offset+paramSizes[6]]
	offset += paramSizes[6]
	model.params.attprojb = paramsMemory[offset : offset+paramSizes[7]]
	offset += paramSizes[7]
	model.params.ln2w = paramsMemory[offset : offset+paramSizes[8]]
	offset += paramSizes[8]
	model.params.ln2b = paramsMemory[offset : offset+paramSizes[9]]
	offset += paramSizes[9]
	model.params.fcw = paramsMemory[offset : offset+paramSizes[10]]
	offset += paramSizes[10]
	model.params.fcb = paramsMemory[offset : offset+paramSizes[11]]
	offset += paramSizes[11]
	model.params.fcprojw = paramsMemory[offset : offset+paramSizes[12]]
	offset += paramSizes[12]
	model.params.fcprojb = paramsMemory[offset : offset+paramSizes[13]]
	offset += paramSizes[13]
	model.params.lnfw = paramsMemory[offset : offset+paramSizes[14]]
	offset += paramSizes[14]
	model.params.lnfb = paramsMemory[offset : offset+paramSizes[15]]

	return nil
}

func gpt2Forward(model *GPT2, inputs []int, B, T int) {
	V := model.config.vocabSize
	L := model.config.numLayers
	NH := model.config.numHeads
	C := model.config.channels

	// 记录当前的batch size和sequence length
	model.batchSize = B
	model.seqLen = T

	// 分配激活值内存
	actSizes := []int{
		B * T * C,          // encoded
		L * B * T * C,      // ln1
		L * B * T,          // ln1_mean
		L * B * T,          // ln1_rstd
		L * B * T * 3 * C,  // qkv
		L * B * T * C,      // atty
		L * B * NH * T * T, // preatt
		L * B * NH * T * T, // att
		L * B * T * C,      // attproj
		L * B * T * C,      // residual2
		L * B * T * C,      // ln2
		L * B * T,          // ln2_mean
		L * B * T,          // ln2_rstd
		L * B * T * 4 * C,  // fch
		L * B * T * 4 * C,  // fch_gelu
		L * B * T * C,      // fcproj
		L * B * T * C,      // residual3
		B * T * C,          // lnf
		B * T,              // lnf_mean
		B * T,              // lnf_rstd
		B * T * V,          // logits
		B * T * V,          // probs
		B * T,              // losses
	}

	numActivations := 0
	for _, size := range actSizes {
		numActivations += size
	}

	actsMemory := make([]float32, numActivations)
	offset := 0
	model.acts.encoded = actsMemory[offset : offset+actSizes[0]]
	offset += actSizes[0]
	model.acts.ln1 = actsMemory[offset : offset+actSizes[1]]
	offset += actSizes[1]
	model.acts.ln1Mean = actsMemory[offset : offset+actSizes[2]]
	offset += actSizes[2]
	model.acts.ln1Rstd = actsMemory[offset : offset+actSizes[3]]
	offset += actSizes[3]
	model.acts.qkv = actsMemory[offset : offset+actSizes[4]]
	offset += actSizes[4]
	model.acts.atty = actsMemory[offset : offset+actSizes[5]]
	offset += actSizes[5]
	model.acts.preatt = actsMemory[offset : offset+actSizes[6]]
	offset += actSizes[6]
	model.acts.att = actsMemory[offset : offset+actSizes[7]]
	offset += actSizes[7]
	model.acts.attproj = actsMemory[offset : offset+actSizes[8]]
	offset += actSizes[8]
	model.acts.residual2 = actsMemory[offset : offset+actSizes[9]]
	offset += actSizes[9]
	model.acts.ln2 = actsMemory[offset : offset+actSizes[10]]
	offset += actSizes[10]
	model.acts.ln2Mean = actsMemory[offset : offset+actSizes[11]]
	offset += actSizes[11]
	model.acts.ln2Rstd = actsMemory[offset : offset+actSizes[12]]
	offset += actSizes[12]
	model.acts.fch = actsMemory[offset : offset+actSizes[13]]
	offset += actSizes[13]
	model.acts.fchGelu = actsMemory[offset : offset+actSizes[14]]
	offset += actSizes[14]
	model.acts.fcproj = actsMemory[offset : offset+actSizes[15]]
	offset += actSizes[15]
	model.acts.residual3 = actsMemory[offset : offset+actSizes[16]]
	offset += actSizes[16]
	model.acts.lnf = actsMemory[offset : offset+actSizes[17]]
	offset += actSizes[17]
	model.acts.lnfMean = actsMemory[offset : offset+actSizes[18]]
	offset += actSizes[18]
	model.acts.lnfRstd = actsMemory[offset : offset+actSizes[19]]
	offset += actSizes[19]
	model.acts.logits = actsMemory[offset : offset+actSizes[20]]
	offset += actSizes[20]
	model.acts.probs = actsMemory[offset : offset+actSizes[21]]
	offset += actSizes[21]
	model.acts.losses = actsMemory[offset : offset+actSizes[22]]

	// forward pass
	params := model.params
	acts := model.acts

	encoderForward(acts.encoded, inputs, params.wte, params.wpe, B, T, C)

	for l := 0; l < L; l++ {
		var residual []float32
		if l == 0 {
			residual = acts.encoded
		} else {
			residual = acts.residual3[(l-1)*B*T*C:]
		}

		// get the pointers of the weights for this layer
		lLn1w := params.ln1w[l*C:]
		lLn1b := params.ln1b[l*C:]
		lQkvw := params.qkvw[l*3*C*C:]
		lQkvb := params.qkvb[l*3*C:]
		lAttprojw := params.attprojw[l*C*C:]
		lAttprojb := params.attprojb[l*C:]
		lLn2w := params.ln2w[l*C:]
		lLn2b := params.ln2b[l*C:]
		lFcw := params.fcw[l*4*C*C:]
		lFcb := params.fcb[l*4*C:]
		lFcprojw := params.fcprojw[l*C*4*C:]
		lFcprojb := params.fcprojb[l*C:]

		// get the pointers of the activations for this layer
		lLn1 := acts.ln1[l*B*T*C:]
		lLn1Mean := acts.ln1Mean[l*B*T:]
		lLn1Rstd := acts.ln1Rstd[l*B*T:]
		lQkv := acts.qkv[l*B*T*3*C:]
		lAtty := acts.atty[l*B*T*C:]
		lPreatt := acts.preatt[l*B*NH*T*T:]
		lAtt := acts.att[l*B*NH*T*T:]
		lAttproj := acts.attproj[l*B*T*C:]
		lResidual2 := acts.residual2[l*B*T*C:]
		lLn2 := acts.ln2[l*B*T*C:]
		lLn2Mean := acts.ln2Mean[l*B*T:]
		lLn2Rstd := acts.ln2Rstd[l*B*T:]
		lFch := acts.fch[l*B*T*4*C:]
		lFchGelu := acts.fchGelu[l*B*T*4*C:]
		lFcproj := acts.fcproj[l*B*T*C:]
		lResidual3 := acts.residual3[l*B*T*C:]

		// now do the forward pass
		layernormForward(lLn1, lLn1Mean, lLn1Rstd, residual, lLn1w, lLn1b, B, T, C)
		matmulForward(lQkv, lLn1, lQkvw, lQkvb, B, T, C, 3*C)
		attentionForward(lAtty, lPreatt, lAtt, lQkv, B, T, C, NH)
		matmulForward(lAttproj, lAtty, lAttprojw, lAttprojb, B, T, C, C)
		residualForward(lResidual2, residual, lAttproj, B*T*C)
		layernormForward(lLn2, lLn2Mean, lLn2Rstd, lResidual2, lLn2w, lLn2b, B, T, C)
		matmulForward(lFch, lLn2, lFcw, lFcb, B, T, C, 4*C)
		geluForward(lFchGelu, lFch, B*T*4*C)
		matmulForward(lFcproj, lFchGelu, lFcprojw, lFcprojb, B, T, 4*C, C)
		residualForward(lResidual3, lResidual2, lFcproj, B*T*C)
	}

	residual := acts.residual3[(L-1)*B*T*C:]
	layernormForward(acts.lnf, acts.lnfMean, acts.lnfRstd, residual, params.lnfw, params.lnfb, B, T, C)
	matmulForward(acts.logits, acts.lnf, params.wte, nil, B, T, C, V)
	softmaxForward(acts.probs, acts.logits, B, T, V)
}

func sampleMult(probabilities []float32, n int) int {
	cdf := float32(0.0)
	coin := float32(0.5)
	for i := 0; i < n; i++ {
		cdf += probabilities[i]
		if coin < cdf {
			return i
		}
	}
	return n - 1
}

const GPT2_EOT = 50256

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Provide at least one token.")
		os.Exit(1)
	}

	const n = 10
	if len(os.Args) > n+1 {
		fmt.Println("Too many tokens.")
		os.Exit(1)
	}

	var model GPT2
	if err := gpt2BuildFromCheckpoint(&model, "gpt2_124M.bin"); err != nil {
		fmt.Printf("Error loading model: %v\n", err)
		os.Exit(1)
	}

	tokens := make([]int, n)
	for i := 0; i < n; i++ {
		if i+1 < len(os.Args) {
			val, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Printf("Invalid token: %s\n", os.Args[i+1])
				os.Exit(1)
			}
			tokens[i] = val
		} else {
			tokens[i] = GPT2_EOT
		}
	}

	for t := len(os.Args) - 1; t < n; t++ {
		gpt2Forward(&model, tokens, 1, t)
		probs := model.acts.probs[(t-1)*model.config.vocabSize : t*model.config.vocabSize]
		nextToken := sampleMult(probs, model.config.vocabSize)
		tokens[t] = nextToken
		fmt.Println(tokens[t])
	}
}
