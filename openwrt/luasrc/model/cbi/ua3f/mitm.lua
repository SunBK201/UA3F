local M = {}

local cbi = require("luci.cbi")
local i18n = require("luci.i18n")
local translate = i18n.translate

local Flag = cbi.Flag
local Value = cbi.Value
local DummyValue = cbi.DummyValue

function M.add_mitm_fields(section)
    -- Enable MitM
    local mitm_enabled = section:taboption("mitm", Flag, "mitm_enabled", translate("Enable HTTPS MitM"))
    mitm_enabled.description = translate(
        "Enable HTTPS Man-in-the-Middle to decrypt and rewrite HTTPS traffic. Requires installing the CA certificate on client devices")

    -- Certificate Status Display
    local cert_status = section:taboption("mitm", DummyValue, "_mitm_cert_status",
        translate("Certificate Status"))
    cert_status.rawhtml = true

    function cert_status.cfgvalue(self, section)
        local uci = require("luci.model.uci").cursor()
        local p12_data = uci:get("ua3f", "main", "mitm_ca_p12_base64")
        if p12_data and p12_data ~= "" then
            return '<input disabled type="button" style="opacity: 1;" class="btn cbi-button cbi-button-add" value="' ..
                translate("CA certificate is configured") .. '" />'
        else
            return '<input disabled type="button" style="opacity: 1;" class="btn cbi-button cbi-button-reset" value="' ..
                translate("No CA certificate. Please generate one.") .. '" />'
        end
    end

    -- Buttons: Generate Certificate & Export Certificate
    local button_container = section:taboption("mitm", DummyValue, "_mitm_button_container",
        translate("Certificate Management"))
    button_container.rawhtml = true

    function button_container.cfgvalue(self, section)
        return ""
    end

    function button_container.render(self, section, scope)
        luci.http.write([[
            <div class="cbi-value" id="cbi-ua3f-main-_mitm_button_container">
                <label class="cbi-value-title">]] .. translate("Certificate Management") .. [[</label>
                <div class="cbi-value-field" style="display: flex; gap: 10px; flex-wrap: wrap;">
                    <input type="button" class="btn cbi-button cbi-button-link"
                           value="]] .. translate("Generate New Certificate") .. [[" id="ua3f-mitm-generate-btn"/>
                    <input type="button" class="btn cbi-button cbi-button-apply"
                           value="]] .. translate("Export Certificate") .. [[" id="ua3f-mitm-export-btn"/>
                </div>
            </div>
            <script>
            (function() {
                var generateBtn = document.getElementById('ua3f-mitm-generate-btn');
                var exportBtn = document.getElementById('ua3f-mitm-export-btn');

                generateBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    if (!confirm(']] ..
            translate(
                "Generate a new CA certificate? The old certificate will be replaced and all clients will need to re-install the new certificate.") ..
            [[')) {
                        return;
                    }
                    var passphrase = prompt(']] ..
            translate("Enter an optional passphrase for the CA certificate (leave empty for none):") .. [[', '');
                    if (passphrase === null) return; // user cancelled
                    generateBtn.disabled = true;
                    generateBtn.value = ']] .. translate("Generating...") .. [[';

                    var xhr = new XMLHttpRequest();
                    xhr.open('POST', ']] ..
            luci.dispatcher.build_url("admin", "services", "ua3f", "mitm_generate_cert") .. [[', true);
                    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');

                    // Get the CSRF token
                    var tokenInput = document.querySelector('input[name="token"]');
                    var token = tokenInput ? tokenInput.value : '';

                    xhr.onload = function() {
                        generateBtn.disabled = false;
                        generateBtn.value = ']] .. translate("Generate New Certificate") .. [[';
                        try {
                            var resp = JSON.parse(xhr.responseText);
                            if (resp.success) {
                                alert(']] ..
            translate("Certificate generated successfully! Please export and install it on client devices.") .. [[');
                                location.reload();
                            } else {
                                alert(']] ..
            translate("Failed to generate certificate") .. [[: ' + (resp.error || 'Unknown error'));
                            }
                        } catch(ex) {
                            alert(']] .. translate("Failed to generate certificate") .. [[: ' + xhr.statusText);
                        }
                    };
                    xhr.onerror = function() {
                        generateBtn.disabled = false;
                        generateBtn.value = ']] .. translate("Generate New Certificate") .. [[';
                        alert(']] .. translate("Network error") .. [[');
                    };
                    xhr.send('token=' + encodeURIComponent(token) + '&passphrase=' + encodeURIComponent(passphrase));
                });

                exportBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    // Trigger download via hidden iframe/link
                    var url = ']] ..
            luci.dispatcher.build_url("admin", "services", "ua3f", "mitm_export_cert") .. [[';
                    var tokenInput = document.querySelector('input[name="token"]');
                    var token = tokenInput ? tokenInput.value : '';
                    window.location.href = url + '?token=' + encodeURIComponent(token);
                });
            })();
            </script>
        ]])
    end

    -- Skip Server Certificate Verification
    local mitm_skip_verify = section:taboption("mitm", Flag, "mitm_skip_verify",
        translate("Skip Server Certificate Verification"))
    mitm_skip_verify.description = translate(
        "Skip verifying the upstream server's TLS certificate during MitM")
    mitm_skip_verify:depends("mitm_enabled", "1")

    -- Hostname
    local mitm_hostname = section:taboption("mitm", Value, "mitm_hostname", translate("MitM Hostname"))
    mitm_hostname.description = translate(
    "Only hosts in this list will be decrypted, supports glob patterns, use :port to specify port, default port is 443, :0 matches all ports")
    mitm_hostname.placeholder = "*.example.com, api.test.com:8443"
    mitm_hostname:depends("mitm_enabled", "1")
end

return M
