# docker model pull

<!---MARKER_GEN_START-->
Pull a model from Docker Hub or HuggingFace to your local environment


<!---MARKER_GEN_END-->

## Description

Pull a model to your local environment. Downloaded models also appear in the Docker Desktop Dashboard.

## Examples

### Pulling a model from Docker Hub

```console
docker model pull ai/smollm2
```

### Pulling from HuggingFace

You can pull GGUF models directly from [Hugging Face](https://huggingface.co/models?library=gguf).

**Note about quantization:** If no tag is specified, the command pulls the non-quantized version of the model.
To specify the quantization, provide the version as a tag, for example:
`docker model pull hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF:Q4_K_S`

```console
docker model pull hf.co/bartowski/Llama-3.2-1B-Instruct-GGUF
```
