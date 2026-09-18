package cmd

import "strings"

// Extend the existing host coordinator only. The existing native host remains
// the sole launcher; these operations prepare metadata pointers and observe
// actual child command/output records. They never author a child ACK.
func nativeContextFetchCoordinator(source string) string {
	const anchor = `if op == "manifest":`
	if strings.Count(source, anchor) != 1 {
		panic("native context fetch coordinator requires one manifest anchor")
	}
	return strings.Replace(source, anchor, nativeContextFetchPython+"\n"+nativeContextFetchOperations+"\nelif op == \"manifest\":", 1)
}

// Pure decoder, with no subprocess, filesystem, environment, or model access.
// Both supported print grammars preserve the actual exec result. All three
// correlated raw host records remain independently available for Go replay.
const nativeContextFetchPython = `
import re, shlex, shutil, urllib.parse
if "scoped" not in globals():
    def scoped(name): return name

def native_fetch_json(raw):
    def pairs(items):
        out = {}
        for key, value in items:
            assert key not in out, "Duplicate JSON key"
            out[key] = value
        return out
    def constant(value): raise AssertionError("Nonfinite JSON number")
    value = json.loads(raw, object_pairs_hook=pairs, parse_constant=constant)
    def valid(node):
        if isinstance(node, str): node.encode("utf-8", "strict")
        elif isinstance(node, list):
            for item in node: valid(item)
        elif isinstance(node, dict):
            for key, item in node.items(): valid(key); valid(item)
    valid(value)
    return value

def native_fetch_hash(raw):
    return "sha256:" + hashlib.sha256(raw).hexdigest()

def native_fetch_time(value):
    match = re.fullmatch(r"(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(\d{1,9}))?(Z|[+-]\d{2}:\d{2})", value or "")
    assert match, "Missing exact host RFC3339 timestamp"
    base = datetime.datetime.fromisoformat(match[1] + ("+00:00" if match[3] == "Z" else match[3]))
    return int(base.timestamp()) * 1000000000 + int((match[2] or "").ljust(9, "0"))

def native_fetch_canonical_time(value):
    seconds, fraction = divmod(native_fetch_time(value), 1000000000)
    base = datetime.datetime.fromtimestamp(seconds, datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%S")
    return base + (("."+str(fraction).zfill(9).rstrip("0")) if fraction else "") + "Z"

def native_fetch_ids(value):
    if value is None: return []
    assert isinstance(value,list) and all(isinstance(x,str) and x.strip() for x in value) and len(set(value)) == len(value), "Invalid exact decision IDs"
    return value

def native_fetch_cwd(value, expected):
    if not isinstance(value, str): return False
    if value.startswith("file://"):
        parsed = urllib.parse.urlsplit(value)
        if parsed.netloc or parsed.query or parsed.fragment: return False
        value = urllib.parse.unquote(parsed.path, errors="strict")
    return value == expected

def native_fetch_literal_object(source):
    # Decode a tiny literal object grammar; never eval/execute JS.
    pos = 0
    def space():
        nonlocal pos
        while pos < len(source) and source[pos].isspace(): pos += 1
    def take(char):
        nonlocal pos
        space()
        assert pos < len(source) and source[pos] == char, "Nonliteral command object"
        pos += 1
    def quoted():
        nonlocal pos
        space()
        assert pos < len(source) and source[pos] in (chr(34), chr(39)), "Expected literal string"
        quote = source[pos]; pos += 1; out = []
        while pos < len(source):
            char = source[pos]; pos += 1
            if char == quote: return "".join(out)
            assert ord(char) >= 32, "Raw control byte in literal string"
            if char != chr(92): out.append(char); continue
            assert pos < len(source), "Incomplete string escape"
            escaped = source[pos]; pos += 1
            simple = {chr(34):chr(34), chr(39):chr(39), chr(92):chr(92), "/":"/", "n":"\n", "r":"\r", "t":"\t", "b":"\b", "f":"\f"}
            if escaped in simple: out.append(simple[escaped]); continue
            assert escaped == "u" and re.fullmatch(r"[0-9a-fA-F]{4}", source[pos:pos+4]), "Unsupported literal escape"
            start = pos - 2; pos += 4
            if 0xD800 <= int(source[pos-4:pos], 16) <= 0xDBFF:
                assert source[pos:pos+2] == chr(92)+"u" and re.fullmatch(r"[0-9a-fA-F]{4}", source[pos+2:pos+6]), "Unpaired Unicode escape"
                pos += 6
            out.append(native_fetch_json(chr(34)+source[start:pos]+chr(34)))
        raise AssertionError("Unterminated literal string")
    take("{"); out = {}
    while True:
        space()
        if pos < len(source) and source[pos] in (chr(34), chr(39)): key = quoted()
        else:
            match = re.match(r"[a-z_]+", source[pos:]); assert match, "Invalid literal key"
            key = match[0]; pos += len(key)
        assert key not in out, "Duplicate literal key"
        take(":"); space()
        if key in ("cmd", "workdir"): out[key] = quoted()
        else:
            assert key in ("yield_time_ms", "max_output_tokens"), "Unsupported exec option"
            match = re.match(r"(?:0|[1-9][0-9]*)", source[pos:]); assert match, "Nonliteral exec option"
            out[key] = int(match[0]); pos += len(match[0])
        space()
        if pos < len(source) and source[pos] == "}": pos += 1; break
        take(",")
    space()
    assert pos == len(source) and isinstance(out.get("cmd"), str), "Incomplete literal exec object"
    return out

def native_fetch_exec(input_text, cwd):
    patterns = [
        (r"\s*text\(await\s+tools\.exec_command\((\{[\s\S]*\})\)\);\s*", "object"),
        (r"\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\((\{[\s\S]*\})\);\s*text\(\1\);\s*", "object"),
        (r"\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\((\{[\s\S]*\})\);\s*text\(\1\.output\);\s*", "output"),
    ]
    for pattern, projection in patterns:
        match = re.fullmatch(pattern, input_text or "")
        if not match: continue
        value = native_fetch_literal_object(match[1] if len(match.groups()) == 1 else match[2])
        assert value.get("workdir", cwd) == cwd, "Child read changed workspace"
        command = value["cmd"]
        assert not any(char in command for char in (";", "&", "|", "<", ">", "$", chr(96), "\n", "\r")), "Unsupported shell syntax"
        argv = shlex.split(command, posix=True)
        return {"command":command, "argv":argv, "projection":projection}
    return None

def native_fetch_source(raw_lines, events, ci, candidate, cwd, child):
    call_event = events[ci]; call = call_event["payload"]
    call_id = call.get("call_id"); turn = call.get("internal_chat_message_metadata_passthrough", {}).get("turn_id")
    assert call_id and turn and call.get("id"), "Missing actual child call identity"
    calls = [n for n,e in enumerate(events) if e.get("type") == "response_item" and e.get("payload", {}).get("type") in ("custom_tool_call", "function_call") and e["payload"].get("call_id") == call_id]
    results = [n for n,e in enumerate(events) if e.get("type") == "response_item" and e.get("payload", {}).get("type") in ("custom_tool_call_output", "function_call_output") and e["payload"].get("call_id") == call_id]
    assert calls == [ci] and len(results) == 1 and results[0] > ci, "Ambiguous or incomplete actual child call/result"
    ri = results[0]; result_event = events[ri]; result = result_event["payload"]
    assert result.get("type") == "custom_tool_call_output" and result.get("id") and result.get("internal_chat_message_metadata_passthrough", {}).get("turn_id") == turn, "Result belongs to another tool/turn"
    executions = []
    for n in range(ci+1, ri):
        event = events[n]; payload = event.get("payload", {}); item = payload.get("item", {})
        assert not (event.get("type") == "response_item" and payload.get("type") in ("custom_tool_call", "function_call")), "Another child call interleaved context fetch"
        if event.get("type") == "event_msg" and payload.get("type") == "item_completed":
            assert item.get("type") != "FileChange", "Child mutation interleaved context fetch"
            if item.get("type") == "CommandExecution": executions.append(n)
    assert len(executions) == 1, "Expected sole actual child CommandExecution"
    ei = executions[0]; command_event = events[ei]; payload = command_event["payload"]; item = payload["item"]
    event_id = item.get("id")
    all_command_ids = [e.get("payload", {}).get("item", {}).get("id") for e in events if e.get("type") == "event_msg" and e.get("payload", {}).get("type") == "item_completed" and e["payload"].get("item", {}).get("type") == "CommandExecution"]
    assert event_id and all_command_ids.count(event_id) == 1, "Duplicated actual command event"
    assert payload.get("thread_id") == child and payload.get("turn_id") == turn, "Command is parent-owned or belongs to another child/turn"
    assert item.get("status") == "completed" and type(item.get("exit_code")) is int and item["exit_code"] == 0, "Child command did not complete successfully"
    assert native_fetch_cwd(item.get("cwd"), cwd), "Host command workspace differs"
    command = item.get("command")
    assert isinstance(command, list) and len(command) == 3 and isinstance(command[0], str) and command[0].startswith("/") and command[1] in ("-c", "-lc") and command[2] == candidate["command"], "Actual host command differs from literal child call"
    stdout = item.get("aggregated_output")
    assert isinstance(stdout, str) and stdout and item.get("stdout") == stdout and item.get("formatted_output") == stdout and item.get("stderr") == "", "Full actual plaintext stdout is unavailable or conflicting"
    output = result.get("output")
    assert isinstance(output, list) and len(output) == 2 and all(isinstance(part, dict) and part.get("type") == "input_text" and isinstance(part.get("text"), str) for part in output), "Incomplete code-mode result"
    assert output[0]["text"].startswith("Script completed\n"), "Interrupted or running code-mode result"
    if candidate["projection"] == "output":
        assert output[1]["text"] == stdout, "Projected output differs from actual stdout"
    else:
        returned = native_fetch_json(output[1]["text"])
        assert isinstance(returned, dict) and type(returned.get("exit_code")) is int and returned["exit_code"] == 0 and "session_id" not in returned and returned.get("output") == stdout, "Full untouched result differs from actual stdout"
    started, executed, completed = [native_fetch_time(e.get("timestamp")) for e in (call_event, command_event, result_event)]
    assert started <= executed <= completed and started < completed, "Host source timestamp order differs"
    def ref(event, raw, identifier): return {"id":identifier, "sha256":hashlib.sha256(raw).hexdigest()}
    source = {"child_id":child,"turn_id":turn,"call_id":call_id,
              "call":ref(call_event,raw_lines[ci],call["id"]),"command":ref(command_event,raw_lines[ei],event_id),"result":ref(result_event,raw_lines[ri],result["id"]),
              "started_at":native_fetch_canonical_time(call_event["timestamp"]),"completed_at":native_fetch_canonical_time(result_event["timestamp"])}
    assert len({source[key]["id"] for key in ("call","command","result")}) == 3, "Source roles share an ambiguous event ID"
    for role in ("call","result"):
        assert sum(e.get("payload",{}).get("id") == source[role]["id"] for e in events) == 1, "Duplicated actual source record ID"
    return {"source":source,"stdout":stdout,"envelope":native_fetch_json(stdout),"indexes":[ci,ei,ri],"raw_lines":[raw_lines[n] for n in (ci,ei,ri)]}

def native_context_fetch_decode(raw, identity, cwd, candidate, requests, reservation):
    assert isinstance(raw, bytes) and raw and candidate.startswith("/"), "Missing child source or absolute candidate"
    raw_lines = raw.splitlines()
    assert raw_lines and all(raw_lines), "Blank or truncated child source record"
    events = [native_fetch_json(line) for line in raw_lines]
    assert all(isinstance(event, dict) for event in events), "Malformed child source envelope"
    meta = events[0].get("payload", {})
    assert events[0].get("type") == "session_meta" and meta.get("id") == identity["child_id"] and meta.get("parent_thread_id") == identity["host_session_id"] and meta.get("agent_path") == identity["task_path"] and meta.get("agent_role") == "aether-" + reservation["worker"]["caste"] and meta.get("cwd") == cwd, "First metadata does not identify exact bound worker"
    assert sum(e.get("type") == "session_meta" for e in events) == 1, "Inherited or duplicate session metadata is outside child-fetch contract"
    paths = {item["path"]:(purpose,item["value"]) for purpose,item in requests.items()}
    reads, acknowledgements = [], []
    for ci,event in enumerate(events):
        call = event.get("payload", {})
        if event.get("type") != "response_item" or call.get("type") != "custom_tool_call": continue
        parsed = None
        try: parsed = native_fetch_exec(call.get("input", ""), cwd)
        except (AssertionError, ValueError):
            if "codex-native-worker" in call.get("input", ""): raise
        if not parsed:
            assert "codex-native-worker" not in call.get("input", ""), "Unclassified runtime child operation"
            continue
        argv = parsed["argv"]
        if len(argv) < 3 or argv[1] != "codex-native-worker": continue
        assert call.get("name") in ("exec","functions.exec") and argv[0] == candidate and argv[2] in ("context","context-ack"), "Wrong candidate or unclassified runtime operation"
        assert len(argv) >= 5 and argv[3] == "--request" and argv[4] in paths, "Child context request is not its immutable metadata pointer"
        purpose, request_value = paths[argv[4]]
        observed = native_fetch_source(raw_lines, events, ci, parsed, cwd, identity["child_id"])
        envelope = observed["envelope"]
        assert isinstance(envelope, dict) and envelope.get("ok") is True and not envelope.get("error") and not envelope.get("code"), "Child received unsuccessful runtime JSON"
        result = envelope.get("result")
        assert isinstance(result, dict) and type(result.get("schema_version")) is int and result["schema_version"] == 1 and result.get("execution_binding") == request_value["execution_binding"] and not result.get("complete") and not result.get("launch_allowed") and not result.get("receipt"), "Read/ACK output has wrong execution binding or claims mutation"
        observed.update(purpose=purpose,request_path=argv[4],argv=argv)
        if argv[2] == "context":
            assert len(argv) == 5 and not result.get("context_ack") and result.get("context_status") == "awaiting_ack", "Read did not await actual child ACK"
            delivery = result.get("context_delivery")
            assert isinstance(delivery, dict) and delivery.get("schema_version") == 2 and delivery.get("protocol") == "child-fetch/v1" and delivery.get("purpose") == purpose, "Wrong context delivery protocol/purpose"
            for key in ("execution_binding","worker_name","task_id","launch_id","host_session_id","child_id","workspace","dispatch_sha256","prompt_sha256"):
                assert delivery.get(key) == request_value[key], "Delivery differs from exact bound request: " + key
            assert isinstance(delivery.get("payload"), str) and delivery["payload"] and delivery.get("payload_sha256") == native_fetch_hash(delivery["payload"].encode("utf-8")), "Exact full Go payload/digest absent"
            assert re.fullmatch(r"(?:sha256:)?[0-9a-f]{64}", delivery.get("delivery_id", "")), "Missing Go delivery identity"
            ids = native_fetch_ids(delivery.get("decision_ids"))
            assert isinstance(ids, list) and all(isinstance(x,str) and x for x in ids) and len(set(ids)) == len(ids), "Invalid decision IDs"
            if purpose == "initial":
                assert delivery["payload"] == reservation["worker"]["native"]["prompt"] and delivery["payload_sha256"] == reservation["worker"]["native"]["prompt_sha256"] and ids == (reservation["worker"]["native"].get("context_decision_ids") or []), "Initial payload is not the exact saved full runtime prompt"
            observed["delivery"] = delivery; reads.append(observed)
        else:
            ack = result.get("context_ack")
            assert not result.get("context_delivery") and result.get("context_status") == "ack_validated" and isinstance(ack,dict) and type(ack.get("schema_version")) is int and ack["schema_version"] == 1 and ack.get("child_id") == identity["child_id"], "Actual runtime child ACK absent"
            ids = native_fetch_ids(ack.get("decision_ids"))
            assert isinstance(ids,list) and all(isinstance(x,str) and x for x in ids) and len(set(ids)) == len(ids), "Invalid ACK decision IDs"
            expected_argv = [candidate,"codex-native-worker","context-ack","--request",argv[4],"--delivery-id",ack.get("delivery_id"),"--payload-sha256",ack.get("payload_sha256")]
            for decision_id in ids: expected_argv += ["--decision-id",decision_id]
            assert argv == expected_argv, "Actual ACK command arguments differ from returned typed ACK"
            observed["ack"] = ack; acknowledgements.append(observed)
    assert reads and acknowledgements, "No completed actual child context read and separate ACK"
    pairs = []
    used = set()
    for read_record in reads:
        delivery = read_record["delivery"]; key = delivery["delivery_id"]
        assert key not in used, "Multiple actual reads create ambiguous source for one delivery"
        used.add(key)
        matches = [a for a in acknowledgements if a["ack"].get("delivery_id") == key]
        assert len(matches) == 1, "Read lacks one distinct completed child ACK"
        ack_record = matches[0]; ack = ack_record["ack"]
        assert ack_record["purpose"] == read_record["purpose"] and ack_record["request_path"] == read_record["request_path"] and ack.get("payload_sha256") == delivery["payload_sha256"] and (ack.get("decision_ids") or []) == (delivery.get("decision_ids") or []), "ACK does not identify exact fetched payload"
        assert read_record["indexes"][2] < ack_record["indexes"][0] and native_fetch_time(read_record["source"]["completed_at"]) < native_fetch_time(ack_record["source"]["started_at"]), "ACK was not separately invoked after completed read"
        refs = [record["source"][role]["id"] for record in (read_record,ack_record) for role in ("call","command","result")]
        assert len(set(refs)) == 6 and read_record["source"]["call_id"] != ack_record["source"]["call_id"], "Read and ACK reuse source identities"
        pairs.append({"delivery":delivery,"ack":ack,"fetch":{"schema_version":1,"read":read_record["source"],"ack":ack_record["source"]},"request_path":read_record["request_path"],"purpose":read_record["purpose"],"read_stdout_sha256":native_fetch_hash(read_record["stdout"].encode("utf-8")),"ack_stdout_sha256":native_fetch_hash(ack_record["stdout"].encode("utf-8")),"raw_lines":read_record["raw_lines"]+ack_record["raw_lines"]})
    assert len(acknowledgements) == len(pairs), "Unmatched actual context ACK"
    source_ids = [pair["fetch"][stage][role]["id"] for pair in pairs for stage in ("read","ack") for role in ("call","command","result")]
    assert len(source_ids) == len(set(source_ids)), "Source record reused across deliveries"
    pairs.sort(key=lambda pair: native_fetch_time(pair["fetch"]["read"]["started_at"]))
    assert pairs[0]["purpose"] == "initial", "Answers cannot precede full initial context"
    for previous,current in zip(pairs,pairs[1:]):
        assert native_fetch_time(previous["fetch"]["ack"]["completed_at"]) < native_fetch_time(current["fetch"]["read"]["started_at"]), "Delivery lifetimes overlap or reorder"
    return pairs
`

const nativeContextFetchOperations = `
def native_fetch_immutable(path, raw):
    if path.exists():
        assert path.read_bytes() == raw, "Refusing to replace retained context evidence"
    else:
        with path.open("xb") as handle: handle.write(raw)

def native_fetch_write(name, value):
    path = coord / scoped(name)
    native_fetch_immutable(path, (json.dumps(value,indent=2,ensure_ascii=False)+"\n").encode("utf-8"))
    return path

def native_fetch_pointer(purpose):
    assert purpose in ("initial","answers")
    value = dict(read("bind-request.json")); value["context_purpose"] = purpose
    assert not any(key in value for key in ("result","context_delivery","context_fetch","context_send","observation_status","context_payload_sha256","context_decision_ids","context_delivery_id")), "Metadata request contains payload/ACK/evidence"
    candidate = shutil.which("aether")
    assert candidate and pathlib.Path(candidate).is_absolute(), "No absolute fixture candidate"
    candidate = str(pathlib.Path(candidate).resolve(strict=True))
    path = native_fetch_write("context-"+purpose+"-request.json",value)
    argv = [candidate,"codex-native-worker","context","--request",str(path)]
    ack = [candidate,"codex-native-worker","context-ack","--request",str(path)]
    pointer = {"schema_version":1,"protocol":"child-fetch/v1","purpose":purpose,"child_id":value["child_id"],"candidate_path":candidate,"candidate_sha256":native_fetch_hash(pathlib.Path(candidate).read_bytes()),"request_path":str(path),"request_sha256":native_fetch_hash(path.read_bytes()),"binding":value,"read_argv":argv,"read_command":shlex.join(argv),"ack_argv_prefix":ack,"ack_command_prefix":shlex.join(ack)}
    native_fetch_write("context-"+purpose+"-pointer.json",pointer)
    return pointer

if op in ("context-initial","context-answers"):
    result = native_fetch_pointer(op.removeprefix("context-"))
elif op == "context-observe":
    value = read("bind-request.json")
    identity = read("child-identity.json")
    assert identity["child_id"] == value["child_id"] and identity["host_session_id"] == value["host_session_id"], "Saved binding and actual child identity differ"
    paths = list(sessions.rglob("*"+identity["child_id"]+".jsonl"))
    assert len(paths) == 1 and paths[0].is_file(), "Expected sole actual bound child rollout"
    child_raw = paths[0].read_bytes()
    parent_paths = list(sessions.rglob("*"+identity["host_session_id"]+".jsonl"))
    assert len(parent_paths) == 1, "Expected sole actual bound parent rollout"
    parent_events = [native_fetch_json(line) for line in parent_paths[0].read_bytes().splitlines()]
    child_metadata = native_fetch_json(child_raw.splitlines()[0])
    actual_identity = resolve_native_child(parent_events,[child_metadata],identity["task_path"],identity["host_session_id"],str(repo))
    assert actual_identity == identity, "Actual parent-child launch mapping changed"
    requests = {}; candidate = None
    for purpose in ("initial","answers"):
        pointer_path = coord / scoped("context-"+purpose+"-pointer.json")
        if not pointer_path.exists(): continue
        pointer = native_fetch_json(pointer_path.read_bytes())
        path = pathlib.Path(pointer["request_path"])
        assert path == coord / scoped("context-"+purpose+"-request.json") and path.is_file(), "Pointer escaped owned request directory"
        request_raw = path.read_bytes(); request_value = native_fetch_json(request_raw)
        assert pointer["request_sha256"] == native_fetch_hash(request_raw) and pointer["binding"] == request_value and request_value == {**value,"context_purpose":purpose}, "Immutable metadata request changed"
        assert pointer["child_id"] == identity["child_id"] and pointer["purpose"] == purpose and pointer["protocol"] == "child-fetch/v1", "Wrong pointer target/purpose"
        assert candidate in (None,pointer["candidate_path"]), "Different candidates across deliveries"
        candidate = pointer["candidate_path"]
        assert candidate == str(pathlib.Path(shutil.which("aether")).resolve(strict=True)) and pointer["candidate_sha256"] == native_fetch_hash(pathlib.Path(candidate).read_bytes()), "Fixture candidate changed"
        requests[purpose] = {"path":str(path),"value":request_value}
    pairs = native_context_fetch_decode(child_raw, identity, str(repo), candidate or "", requests, read("reservation.json"))
    observed = []
    for pair in pairs:
        delivery = pair["delivery"]; fetch = pair["fetch"]
        observation = {**value,"observation_status":"context_fetched","observed_at":fetch["ack"]["completed_at"],"source_event_id":fetch["ack"]["result"]["id"],"source_event_sha256":fetch["ack"]["result"]["sha256"],"context_delivery":delivery,"context_fetch":fetch}
        label = "context-fetch-"+pair["purpose"]+"-"+delivery["delivery_id"].removeprefix("sha256:")
        request_path = native_fetch_write(label+"-observe-request.json",observation)
        source_path = coord / scoped(label+"-sources.jsonl")
        native_fetch_immutable(source_path,b"\n".join(pair["raw_lines"])+b"\n")
        provenance = {key:item for key,item in pair.items() if key != "raw_lines"}
        provenance.update(child_rollout=str(paths[0]),source_records=str(source_path),source_records_sha256=native_fetch_hash(source_path.read_bytes()),candidate=candidate)
        native_fetch_write(label+"-provenance.json",provenance)
        output_path = coord / (scoped(label)+".stdout.json")
        if output_path.exists():
            envelope = native_fetch_json(output_path.read_bytes())
            assert envelope.get("ok") is True and envelope["result"].get("receipt",{}).get("context_delivery_id") == delivery["delivery_id"], "Retained original observation is not successful exact delivery"
            accepted = envelope["result"]
        else:
            accepted = runtime(["codex-native-worker","observe","--request",str(request_path)],label)
        observed.append({"purpose":pair["purpose"],"delivery_id":delivery["delivery_id"],"child_id":identity["child_id"],"receipt":accepted.get("receipt"),"source_records":str(source_path)})
    result = {"protocol":"child-fetch/v1","observed":observed}
`
