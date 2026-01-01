package builtin

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/anthropics/slop/internal/evaluator"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func (r *Registry) registerGeneratorFunctions() {
	// Random
	r.Register("random_seed", builtinRandomSeed)
	r.Register("random_int", builtinRandomInt)
	r.Register("random_float", builtinRandomFloat)
	r.Register("random_choice", builtinRandomChoice)
	r.Register("random_choices", builtinRandomChoices)
	r.Register("random_shuffle", builtinRandomShuffle)
	r.Register("random_chance", builtinRandomChance)
	r.Register("random_weighted", builtinRandomWeighted)
	r.Register("random_uuid", builtinRandomUuid)
	r.Register("random_hex", builtinRandomHex)

	// Generators
	r.Register("gen_name", builtinGenName)
	r.Register("gen_first_name", builtinGenFirstName)
	r.Register("gen_last_name", builtinGenLastName)
	r.Register("gen_email", builtinGenEmail)
	r.Register("gen_phone", builtinGenPhone)
	r.Register("gen_word", builtinGenWord)
	r.Register("gen_words", builtinGenWords)
	r.Register("gen_sentence", builtinGenSentence)
	r.Register("gen_paragraph", builtinGenParagraph)
	r.Register("gen_lorem", builtinGenLorem)
	r.Register("gen_color", builtinGenColor)
	r.Register("gen_rgb", builtinGenRgb)
}

// Random functions

func builtinRandomSeed(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_seed", args, 1); err != nil {
		return nil, err
	}

	seed, err := requireInt("random_seed", args[0])
	if err != nil {
		return nil, err
	}

	rng = rand.New(rand.NewSource(seed))
	return evaluator.NONE, nil
}

func builtinRandomInt(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_int", args, 2); err != nil {
		return nil, err
	}

	minVal, err := requireInt("random_int", args[0])
	if err != nil {
		return nil, err
	}

	maxVal, err := requireInt("random_int", args[1])
	if err != nil {
		return nil, err
	}

	if minVal > maxVal {
		return nil, fmt.Errorf("random_int() min must be <= max")
	}

	val := minVal + rng.Int63n(maxVal-minVal+1)
	return &evaluator.IntValue{Value: val}, nil
}

func builtinRandomFloat(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_float", args, 2); err != nil {
		return nil, err
	}

	minVal, err := requireFloat("random_float", args[0])
	if err != nil {
		return nil, err
	}

	maxVal, err := requireFloat("random_float", args[1])
	if err != nil {
		return nil, err
	}

	if minVal > maxVal {
		return nil, fmt.Errorf("random_float() min must be <= max")
	}

	val := minVal + rng.Float64()*(maxVal-minVal)
	return &evaluator.FloatValue{Value: val}, nil
}

func builtinRandomChoice(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_choice", args, 1); err != nil {
		return nil, err
	}

	list, err := requireList("random_choice", args[0])
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("random_choice() cannot choose from empty list")
	}

	idx := rng.Intn(len(list))
	return list[idx], nil
}

func builtinRandomChoices(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_choices", args, 2); err != nil {
		return nil, err
	}

	list, err := requireList("random_choices", args[0])
	if err != nil {
		return nil, err
	}

	n, err := requireInt("random_choices", args[1])
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("random_choices() cannot choose from empty list")
	}

	result := make([]evaluator.Value, n)
	for i := range result {
		idx := rng.Intn(len(list))
		result[i] = list[idx]
	}

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinRandomShuffle(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_shuffle", args, 1); err != nil {
		return nil, err
	}

	list, err := requireList("random_shuffle", args[0])
	if err != nil {
		return nil, err
	}

	// Make a copy
	result := make([]evaluator.Value, len(list))
	copy(result, list)

	// Shuffle
	rng.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})

	return &evaluator.ListValue{Elements: result}, nil
}

func builtinRandomChance(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_chance", args, 1); err != nil {
		return nil, err
	}

	p, err := requireFloat("random_chance", args[0])
	if err != nil {
		return nil, err
	}

	if p < 0 || p > 1 {
		return nil, fmt.Errorf("random_chance() probability must be between 0 and 1")
	}

	if rng.Float64() < p {
		return evaluator.TRUE, nil
	}
	return evaluator.FALSE, nil
}

func builtinRandomWeighted(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_weighted", args, 1); err != nil {
		return nil, err
	}

	weights, err := requireMap("random_weighted", args[0])
	if err != nil {
		return nil, err
	}

	if len(weights) == 0 {
		return nil, fmt.Errorf("random_weighted() cannot choose from empty map")
	}

	// Calculate total weight
	totalWeight := 0.0
	choices := make([]struct {
		key    string
		weight float64
	}, 0, len(weights))

	for k, v := range weights {
		w, ok := toFloat(v)
		if !ok {
			return nil, fmt.Errorf("random_weighted() weights must be numeric")
		}
		if w < 0 {
			return nil, fmt.Errorf("random_weighted() weights must be non-negative")
		}
		totalWeight += w
		choices = append(choices, struct {
			key    string
			weight float64
		}{k, w})
	}

	if totalWeight == 0 {
		return nil, fmt.Errorf("random_weighted() total weight cannot be zero")
	}

	// Select random value
	r := rng.Float64() * totalWeight
	cumulative := 0.0
	for _, c := range choices {
		cumulative += c.weight
		if r <= cumulative {
			return &evaluator.StringValue{Value: c.key}, nil
		}
	}

	// Fallback to last choice (shouldn't happen)
	return &evaluator.StringValue{Value: choices[len(choices)-1].key}, nil
}

func builtinRandomUuid(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_uuid", args, 0); err != nil {
		return nil, err
	}

	b := make([]byte, 16)
	rng.Read(b)

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
	return &evaluator.StringValue{Value: uuid}, nil
}

func builtinRandomHex(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("random_hex", args, 1); err != nil {
		return nil, err
	}

	n, err := requireInt("random_hex", args[0])
	if err != nil {
		return nil, err
	}

	if n < 0 {
		return nil, fmt.Errorf("random_hex() length must be non-negative")
	}

	const hexChars = "0123456789abcdef"
	result := make([]byte, n)
	for i := range result {
		result[i] = hexChars[rng.Intn(len(hexChars))]
	}

	return &evaluator.StringValue{Value: string(result)}, nil
}

// Generator functions

var firstNames = []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
	"William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Emma", "Oliver", "Ava", "Liam", "Sophia", "Noah"}

var lastNames = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White", "Harris"}

var loremWords = []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
	"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore", "magna", "aliqua",
	"enim", "ad", "minim", "veniam", "quis", "nostrud", "exercitation", "ullamco", "laboris", "nisi",
	"aliquip", "ex", "ea", "commodo", "consequat", "duis", "aute", "irure", "in", "reprehenderit",
	"voluptate", "velit", "esse", "cillum", "fugiat", "nulla", "pariatur", "excepteur", "sint",
	"occaecat", "cupidatat", "non", "proident", "sunt", "culpa", "qui", "officia", "deserunt"}

var emailDomains = []string{"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "example.com"}

func builtinGenName(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_name", args, 0); err != nil {
		return nil, err
	}

	// Could use gender kwarg to filter, but for simplicity we just pick random
	first := firstNames[rng.Intn(len(firstNames))]
	last := lastNames[rng.Intn(len(lastNames))]

	return &evaluator.StringValue{Value: first + " " + last}, nil
}

func builtinGenFirstName(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_first_name", args, 0); err != nil {
		return nil, err
	}

	return &evaluator.StringValue{Value: firstNames[rng.Intn(len(firstNames))]}, nil
}

func builtinGenLastName(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_last_name", args, 0); err != nil {
		return nil, err
	}

	return &evaluator.StringValue{Value: lastNames[rng.Intn(len(lastNames))]}, nil
}

func builtinGenEmail(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_email", args, 0); err != nil {
		return nil, err
	}

	first := strings.ToLower(firstNames[rng.Intn(len(firstNames))])
	last := strings.ToLower(lastNames[rng.Intn(len(lastNames))])
	domain := emailDomains[rng.Intn(len(emailDomains))]

	email := fmt.Sprintf("%s.%s@%s", first, last, domain)
	return &evaluator.StringValue{Value: email}, nil
}

func builtinGenPhone(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_phone", args, 0); err != nil {
		return nil, err
	}

	// Generate US-style phone number
	area := 200 + rng.Intn(800)
	prefix := 200 + rng.Intn(800)
	line := rng.Intn(10000)

	phone := fmt.Sprintf("(%03d) %03d-%04d", area, prefix, line)
	return &evaluator.StringValue{Value: phone}, nil
}

func builtinGenWord(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_word", args, 0); err != nil {
		return nil, err
	}

	return &evaluator.StringValue{Value: loremWords[rng.Intn(len(loremWords))]}, nil
}

func builtinGenWords(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_words", args, 1); err != nil {
		return nil, err
	}

	n, err := requireInt("gen_words", args[0])
	if err != nil {
		return nil, err
	}

	words := make([]string, n)
	for i := range words {
		words[i] = loremWords[rng.Intn(len(loremWords))]
	}

	return &evaluator.StringValue{Value: strings.Join(words, " ")}, nil
}

func builtinGenSentence(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_sentence", args, 0); err != nil {
		return nil, err
	}

	wordCount := 5 + rng.Intn(10)
	words := make([]string, wordCount)
	for i := range words {
		words[i] = loremWords[rng.Intn(len(loremWords))]
	}

	sentence := strings.Join(words, " ")
	sentence = strings.ToUpper(string(sentence[0])) + sentence[1:] + "."

	return &evaluator.StringValue{Value: sentence}, nil
}

func builtinGenParagraph(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_paragraph", args, 0); err != nil {
		return nil, err
	}

	sentenceCount := 3 + rng.Intn(5)
	sentences := make([]string, sentenceCount)

	for i := range sentences {
		wordCount := 5 + rng.Intn(10)
		words := make([]string, wordCount)
		for j := range words {
			words[j] = loremWords[rng.Intn(len(loremWords))]
		}
		sentence := strings.Join(words, " ")
		sentences[i] = strings.ToUpper(string(sentence[0])) + sentence[1:] + "."
	}

	return &evaluator.StringValue{Value: strings.Join(sentences, " ")}, nil
}

func builtinGenLorem(args []evaluator.Value, kwargs map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_lorem", args, 0); err != nil {
		return nil, err
	}

	wordCount := int64(100)
	if wc, ok := kwargs["words"]; ok {
		if iv, ok := wc.(*evaluator.IntValue); ok {
			wordCount = iv.Value
		}
	}

	words := make([]string, wordCount)
	for i := range words {
		words[i] = loremWords[rng.Intn(len(loremWords))]
	}

	// Capitalize first word and add periods
	text := strings.Join(words, " ")
	text = strings.ToUpper(string(text[0])) + text[1:]

	return &evaluator.StringValue{Value: text}, nil
}

func builtinGenColor(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_color", args, 0); err != nil {
		return nil, err
	}

	r := rng.Intn(256)
	g := rng.Intn(256)
	b := rng.Intn(256)

	color := fmt.Sprintf("#%02x%02x%02x", r, g, b)
	return &evaluator.StringValue{Value: color}, nil
}

func builtinGenRgb(args []evaluator.Value, _ map[string]evaluator.Value) (evaluator.Value, error) {
	if err := requireArgs("gen_rgb", args, 0); err != nil {
		return nil, err
	}

	r := rng.Intn(256)
	g := rng.Intn(256)
	b := rng.Intn(256)

	return &evaluator.ListValue{
		Elements: []evaluator.Value{
			&evaluator.IntValue{Value: int64(r)},
			&evaluator.IntValue{Value: int64(g)},
			&evaluator.IntValue{Value: int64(b)},
		},
	}, nil
}
