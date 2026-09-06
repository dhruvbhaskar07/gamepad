// ========================================================
// USER PROFILE & LAYOUT PERSISTENCE ENGINE
// ========================================================
export class ProfileManager {
    static STORAGE_PROFILES_KEY = 'vibe_user_profiles_v1';
    static STORAGE_ACTIVE_KEY = 'vibe_active_profile_id_v1';

    static getDefaultProfiles() {
        return {
            'default': {
                id: 'default',
                name: 'Default Profile',
                created: Date.now(),
                isDefault: true,
                settings: {
                    profile: 'ps5',
                    slot: 1,
                    deadzone: 0.05,
                    stickSens: 1.0,
                    hapticIntensity: 40,
                    gyroEnabled: false,
                    gyroSens: 1.2,
                    invertY: false,
                    themeCat: 'ps',
                    accentColor: '#00d4ff',
                    globalScale: 100,
                    bgFx: 'aurora',
                    gfxIntensity: 85,
                    showGlyphs: true,
                    positions: {}
                }
            }
        };
    }

    static loadAllProfiles() {
        try {
            const raw = localStorage.getItem(this.STORAGE_PROFILES_KEY);
            if (!raw) return this.getDefaultProfiles();
            const parsed = JSON.parse(raw);
            if (!parsed.default) parsed.default = this.getDefaultProfiles().default;
            return parsed;
        } catch (e) {
            return this.getDefaultProfiles();
        }
    }

    static saveAllProfiles(profiles) {
        try {
            localStorage.setItem(this.STORAGE_PROFILES_KEY, JSON.stringify(profiles));
        } catch (e) {}
    }

    static getActiveProfileId() {
        return localStorage.getItem(this.STORAGE_ACTIVE_KEY) || 'default';
    }

    static setActiveProfileId(id) {
        localStorage.setItem(this.STORAGE_ACTIVE_KEY, id);
    }

    static getActiveProfile() {
        const profiles = this.loadAllProfiles();
        const activeId = this.getActiveProfileId();
        return profiles[activeId] || profiles['default'] || Object.values(profiles)[0];
    }

    static saveActiveProfileSettings(settings) {
        const profiles = this.loadAllProfiles();
        const activeId = this.getActiveProfileId();
        if (profiles[activeId]) {
            profiles[activeId].settings = {
                ...profiles[activeId].settings,
                ...settings
            };
            this.saveAllProfiles(profiles);
        }
    }

    static createProfile(name, baseSettings = null) {
        const profiles = this.loadAllProfiles();
        const id = 'prof_' + Date.now();
        profiles[id] = {
            id: id,
            name: name.trim() || 'New Preset',
            created: Date.now(),
            settings: baseSettings ? JSON.parse(JSON.stringify(baseSettings)) : { ...this.getDefaultProfiles().default.settings }
        };
        this.saveAllProfiles(profiles);
        this.setActiveProfileId(id);
        return id;
    }

    static deleteProfile(id) {
        if (id === 'default') return false;
        const profiles = this.loadAllProfiles();
        delete profiles[id];
        this.saveAllProfiles(profiles);
        if (this.getActiveProfileId() === id) {
            this.setActiveProfileId('default');
        }
        return true;
    }

    static exportProfilesJSON() {
        const profiles = this.loadAllProfiles();
        const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(profiles, null, 2));
        const a = document.createElement('a');
        a.href = dataStr;
        a.download = `dualsense_profiles_${new Date().toISOString().slice(0, 10)}.json`;
        document.body.appendChild(a);
        a.click();
        a.remove();
    }

    static importProfilesJSON(jsonText) {
        try {
            const imported = JSON.parse(jsonText);
            if (typeof imported !== 'object') throw new Error('Invalid JSON');
            const current = this.loadAllProfiles();
            const merged = { ...current, ...imported };
            this.saveAllProfiles(merged);
            return true;
        } catch (e) {
            return false;
        }
    }
}
