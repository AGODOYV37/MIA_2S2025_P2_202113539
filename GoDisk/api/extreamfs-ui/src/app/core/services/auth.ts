import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable, throwError, map, tap } from 'rxjs';
import { Commands, ExecResponse } from './commands';

export interface Session {
  user: string;
  mountId: string;   // ID de partición (p.ej. 39A1)
  isRoot: boolean;
}

const LS_KEY = 'extreamfs.session';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private api = inject(Commands);
  private _session$ = new BehaviorSubject<Session | null>(this.restore());
  readonly session$ = this._session$.asObservable();

  get session(): Session | null { return this._session$.value; }
  get isLoggedIn(): boolean { return !!this._session$.value; }

  /** Intenta login vía /api/exec -> "login -usr= -pwd= -id=" */
  login(usr: string, pwd: string, mountId: string): Observable<Session> {
    const script = `login -usr="${usr}" -pwd="${pwd}" -id="${mountId}"`;

    return this.api.execute(script).pipe(
      map((res: ExecResponse) => {
        const out = (res?.output || '').trim();

        // Heurística robusta: si aparece "Error" en cualquier forma => fallo
        const hasError = /(^|\n)\s*error[:\s]/i.test(out) || /(^|\n)\s*Error[:\s]/.test(out);
        if (!out || hasError) {
          const msg = out || 'Error desconocido de autenticación';
          throw new Error(msg);
        }

        // Éxito: persistimos sesión en memoria + localStorage
        const sess: Session = {
          user: usr,
          mountId,
          isRoot: usr.trim().toLowerCase() === 'root',
        };
        this.persist(sess);
        return sess;
      }),
      tap(sess => this._session$.next(sess))
    );
  }

  /** Llama logout por /api/exec y limpia el estado local. */
  logout(): Observable<void> {
    return this.api.execute('logout').pipe(
      map(() => {
        this.clear();
      })
    );
  }

  private persist(s: Session) {
    try { localStorage.setItem(LS_KEY, JSON.stringify(s)); } catch {}
  }
  private restore(): Session | null {
    try {
      const raw = localStorage.getItem(LS_KEY);
      if (!raw) return null;
      const parsed = JSON.parse(raw);
      if (parsed && parsed.user && parsed.mountId) return parsed as Session;
    } catch {}
    return null;
  }
  private clear() {
    this._session$.next(null);
    try { localStorage.removeItem(LS_KEY); } catch {}
  }
}
