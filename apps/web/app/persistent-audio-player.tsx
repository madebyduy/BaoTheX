"use client";

import Link from "next/link";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

export type AudioTrack = {
  id: string;
  src: string;
  title: string;
  subtitle?: string;
  href?: string;
};

type AudioPlayerValue = {
  track: AudioTrack | null;
  playing: boolean;
  playTrack: (track: AudioTrack) => void;
  toggle: () => void;
};

const AudioPlayerContext = createContext<AudioPlayerValue | null>(null);
const STORAGE_KEY = "baothex:audio-player";

export function PersistentAudioProvider({ children }: { children: React.ReactNode }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const resumeAtRef = useRef(0);
  const lastStoredSecondRef = useRef(-1);
  const [track, setTrack] = useState<AudioTrack | null>(null);
  const [playing, setPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);

  const startPlayback = useCallback((audio: HTMLAudioElement) => {
    void audio.play().catch(() => setPlaying(false));
  }, []);

  const persist = useCallback((nextTrack: AudioTrack | null, time: number) => {
    if (!nextTrack) {
      localStorage.removeItem(STORAGE_KEY);
      return;
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ track: nextTrack, currentTime: time }));
  }, []);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const saved = JSON.parse(raw) as { track?: AudioTrack; currentTime?: number };
      if (!saved.track?.src || !saved.track.id) return;
      resumeAtRef.current = Math.max(0, Number(saved.currentTime) || 0);
      setTrack(saved.track);
      if (audioRef.current) {
        audioRef.current.src = saved.track.src;
        audioRef.current.load();
      }
    } catch {
      localStorage.removeItem(STORAGE_KEY);
    }
  }, []);

  const playTrack = useCallback(
    (nextTrack: AudioTrack) => {
      const audio = audioRef.current;
      if (!audio) return;
      if (track?.id === nextTrack.id) {
        if (audio.paused) startPlayback(audio);
        else audio.pause();
        return;
      }
      resumeAtRef.current = 0;
      setTrack(nextTrack);
      setCurrentTime(0);
      setDuration(0);
      audio.src = nextTrack.src;
      audio.load();
      persist(nextTrack, 0);
      startPlayback(audio);
    },
    [persist, startPlayback, track?.id],
  );

  const toggle = useCallback(() => {
    const audio = audioRef.current;
    if (!audio || !track) return;
    if (audio.paused) startPlayback(audio);
    else audio.pause();
  }, [startPlayback, track]);

  const seek = useCallback(
    (seconds: number) => {
      const audio = audioRef.current;
      if (!audio || !Number.isFinite(seconds)) return;
      audio.currentTime = Math.max(0, Math.min(seconds, audio.duration || seconds));
      setCurrentTime(audio.currentTime);
      persist(track, audio.currentTime);
    },
    [persist, track],
  );

  const close = useCallback(() => {
    const audio = audioRef.current;
    if (audio) {
      audio.pause();
      audio.removeAttribute("src");
      audio.load();
    }
    setTrack(null);
    setPlaying(false);
    setCurrentTime(0);
    setDuration(0);
    persist(null, 0);
  }, [persist]);

  const value = useMemo(
    () => ({ track, playing, playTrack, toggle }),
    [playTrack, playing, toggle, track],
  );

  return (
    <AudioPlayerContext.Provider value={value}>
      {children}
      <audio
        ref={audioRef}
        preload="metadata"
        onLoadedMetadata={(event) => {
          const audio = event.currentTarget;
          setDuration(Number.isFinite(audio.duration) ? audio.duration : 0);
          if (resumeAtRef.current > 0 && resumeAtRef.current < audio.duration) {
            audio.currentTime = resumeAtRef.current;
            setCurrentTime(resumeAtRef.current);
            resumeAtRef.current = 0;
          }
        }}
        onPlay={() => setPlaying(true)}
        onPause={(event) => {
          setPlaying(false);
          persist(track, event.currentTarget.currentTime);
        }}
        onTimeUpdate={(event) => {
          const time = event.currentTarget.currentTime;
          setCurrentTime(time);
          const second = Math.floor(time);
          if (second % 5 === 0 && second !== lastStoredSecondRef.current) {
            lastStoredSecondRef.current = second;
            persist(track, time);
          }
        }}
        onEnded={() => {
          setPlaying(false);
          setCurrentTime(0);
          persist(track, 0);
        }}
      />
      {track ? (
        <aside className="persistent-audio-player" aria-label="Trình phát audio BaoTheX">
          <button
            className="audio-skip"
            type="button"
            onClick={() => seek(currentTime - 15)}
            aria-label="Lùi 15 giây"
          >
            −15
          </button>
          <button
            className="audio-toggle"
            type="button"
            onClick={toggle}
            aria-label={playing ? "Tạm dừng" : "Phát"}
          >
            {playing ? "Ⅱ" : "▶"}
          </button>
          <button
            className="audio-skip"
            type="button"
            onClick={() => seek(currentTime + 15)}
            aria-label="Tiến 15 giây"
          >
            +15
          </button>
          <div className="audio-now-playing">
            <span>ĐANG NGHE</span>
            {track.href ? (
              <Link href={track.href}>{track.title}</Link>
            ) : (
              <strong>{track.title}</strong>
            )}
            <small>{track.subtitle || "Báo Thể Ích"}</small>
          </div>
          <div className="audio-progress-wrap">
            <input
              aria-label="Tiến độ audio"
              type="range"
              min="0"
              max={Math.max(duration, 1)}
              step="1"
              value={Math.min(currentTime, Math.max(duration, 1))}
              onChange={(event) => seek(Number(event.target.value))}
            />
            <span>
              {formatTime(currentTime)} / {formatTime(duration)}
            </span>
          </div>
          <button
            className="audio-close"
            type="button"
            onClick={close}
            aria-label="Đóng trình phát"
          >
            ×
          </button>
        </aside>
      ) : null}
    </AudioPlayerContext.Provider>
  );
}

export function usePersistentAudio() {
  const value = useContext(AudioPlayerContext);
  if (!value) throw new Error("usePersistentAudio must be used inside PersistentAudioProvider");
  return value;
}

function formatTime(value: number) {
  if (!Number.isFinite(value) || value < 0) return "0:00";
  const minutes = Math.floor(value / 60);
  const seconds = Math.floor(value % 60);
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}
