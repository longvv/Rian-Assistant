import urllib.request, json
url = "https://openrouter.ai/api/v1/models"
req = urllib.request.Request(url)
with urllib.request.urlopen(req) as response:
    data = json.loads(response.read())
    for model in data['data']:
        if model['id'].endswith(':free'):
            # check the description for "tool" or check pricing
            print(f"- {model['id']} | Tools: {model.get('architecture', {}).get('instruct_type', '')}")
