from flask import Flask, request, jsonify, render_template

app = Flask(__name__)

token_used = []



@app.route('/verify_token', methods=['POST'])
def verify_token():
    data = request.get_json()
    token = data.get('token', '')
    id = data.get('id', '')
    print("ID: {0}".format(id))
    if token not in token_used:
        token_used.append(token)
        return jsonify({'status': 'OK'})
    else:
        return jsonify({'error': 'Unauthorized'}), 401

@app.route('/', methods=['GET'])
def index():
    print(token_used)
    token = 'huhuhuhu'  # Replace with the actual token data
    return render_template('webrtc_index.html', token=token)




if __name__ == '__main__':
    app.run(host="0.0.0.0", port=5000, debug=True)