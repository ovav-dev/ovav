const CIMA = Object.freeze({
  SHEETS: {
    CIMA: 'CIMA',
    PERSONAL: 'PERSONAL',
    OPERACIONES: 'OPERACIONES',
    CONFIG: 'CONFIG',
  },
  ROWS: {
    PERSONAL_HEADER: 4,
    PERSONAL_START: 5,
    OPS_HEADER: 4,
    OPS_START: 5,
  },
  STATUS: Object.freeze({
    PENDING: 'PENDING',
    APPROVED: 'APPROVED',
    REJECTED: 'REJECTED',
    CANCELLED: 'CANCELLED',
    AUTO_REJECTED: 'AUTO_REJECTED',
  }),
  TYPES: Object.freeze({
    OT: 'HORAS_EXTRA',
    SHIFT: 'CAMBIO_HORARIO',
    SWAP: 'INTERCAMBIO_DESCANSO',
    ADMIN: 'ADMINISTRATIVO',
  }),
  PERSONAL: Object.freeze({
    DNI: 1, NAME: 2, PHONE: 3, PERSONAL_EMAIL: 4, CORP_EMAIL: 5,
    SCHEDULE: 6, JOIN_DATE: 7, MODALITY: 8, ADMIN_STATUS: 9, SUPERVISOR: 10,
    TURN_ID: 11, START: 12, END: 13, SEGMENT: 14, COMPLETENESS: 15, CIMA_STATUS: 16,
  }),
  OPS: Object.freeze({
    REQUEST_ID: 1, DATE: 2, TYPE: 3, DNI: 4, NAME: 5, SCHEDULE: 6, TURN_ID: 7,
    SEGMENT: 8, MODALITY: 9, PRE_POST: 10, HOURS: 11, NEW_SCHEDULE: 12, REASON: 13,
    COUNTERPART: 14, COMPANY: 15, CURRENT_DAY_OFF: 16, NEW_DAY_OFF: 17,
    ADMIN_EVENT: 18, STATUS: 19, CREATED_AT: 20, CREATED_BY: 21, UPDATED_AT: 22,
    UPDATED_BY: 23, OBSERVATION: 24, GTR_EXPORT: 25, DETAIL: 26,
  }),
  PROPS: { SPREADSHEET_ID: 'CIMA_SPREADSHEET_ID', LAST_CLOSE: 'CIMA_LAST_CLOSE' },
});

function onOpen() {
  SpreadsheetApp.getUi()
    .createMenu('CIMA')
    .addItem('Abrir panel operativo', 'showCimaSidebar')
    .addItem('Abrir administración', 'showAdminSidebar')
    .addSeparator()
    .addItem('Actualizar CIMA', 'refreshDashboard')
    .addItem('Diagnóstico', 'diagnoseCima')
    .addItem('Instalar / reparar triggers', 'installCima')
    .addSeparator()
    .addItem('Generar GTR de aprobados', 'exportApprovedGtr')
    .addToUi();
}

function installCima() {
  const ss = SpreadsheetApp.getActiveSpreadsheet();
  if (!ss) throw new Error('Abre este proyecto desde el archivo CIMA.');
  PropertiesService.getScriptProperties().setProperty(CIMA.PROPS.SPREADSHEET_ID, ss.getId());
  installTriggers_();
  refreshDashboard();
  ss.toast('CIMA instalado y listo.', 'CIMA', 4);
}

function showCimaSidebar() {
  const html = HtmlService.createTemplateFromFile('Index').evaluate()
    .setTitle('CIMA | Operación');
  SpreadsheetApp.getUi().showSidebar(html);
}

function showAdminSidebar() {
  const html = HtmlService.createTemplateFromFile('Admin').evaluate()
    .setTitle('CIMA | Administración');
  SpreadsheetApp.getUi().showSidebar(html);
}

function include(name) {
  return HtmlService.createHtmlOutputFromFile(name).getContent();
}

function getBook_() {
  const id = PropertiesService.getScriptProperties().getProperty(CIMA.PROPS.SPREADSHEET_ID);
  if (!id) throw new Error('CIMA aún no está instalado. Ejecuta installCima().');
  return SpreadsheetApp.openById(id);
}

function sheet_(name) {
  const sh = getBook_().getSheetByName(name);
  if (!sh) throw new Error(`No existe la hoja ${name}.`);
  return sh;
}

function now_() {
  return new Date();
}

function norm_(value) {
  return String(value ?? '').trim();
}

function normalizeDni_(value) {
  return norm_(value).replace(/\D/g, '');
}

function isValidDni_(dni) {
  return /^\d{8}$/.test(normalizeDni_(dni));
}

function isValidEmail_(email) {
  const value = norm_(email);
  return value === '' || /^[^@\s]+@[^@\s]+$/.test(value);
}

function getCurrentActor_() {
  let active = '';
  try { active = norm_(Session.getActiveUser().getEmail()); } catch (_) {}
  return active || 'GOOGLE_USER';
}

function getConfig_() {
  const values = sheet_(CIMA.SHEETS.CONFIG).getRange('A4:C13').getValues();
  const map = {};
  values.forEach(r => { if (norm_(r[0])) map[norm_(r[0])] = r[1]; });
  return map;
}

function getPolicy_(name) {
  const values = sheet_(CIMA.SHEETS.CONFIG).getRange('E4:G8').getValues();
  const row = values.find(r => norm_(r[0]) === name);
  return row ? row[1] : null;
}

function getTurnCatalog_() {
  return sheet_(CIMA.SHEETS.CONFIG).getRange('I4:L13').getValues()
    .filter(r => norm_(r[0]))
    .map(r => ({ id: norm_(r[0]), schedule: norm_(r[1]), start: r[2], end: r[3] }));
}

function getOperationTypes_() {
  return sheet_(CIMA.SHEETS.CONFIG).getRange('E12:G15').getValues()
    .filter(r => norm_(r[0]) && String(r[1]) !== '0')
    .map(r => ({ type: norm_(r[0]), description: norm_(r[2]) }));
}


/*******************************************************
 * 10_Data.gs
 *******************************************************/

function findPersonalByDni_(dni) {
  const key = normalizeDni_(dni);
  if (!isValidDni_(key)) return null;
  const sh = sheet_(CIMA.SHEETS.PERSONAL);
  const last = Math.max(sh.getLastRow(), CIMA.ROWS.PERSONAL_HEADER);
  const rows = last < CIMA.ROWS.PERSONAL_START ? [] : sh.getRange(CIMA.ROWS.PERSONAL_START, 1, last - CIMA.ROWS.PERSONAL_HEADER, 16).getValues();
  for (let i = 0; i < rows.length; i++) {
    if (normalizeDni_(rows[i][CIMA.PERSONAL.DNI - 1]) === key) {
      return personalRowToObject_(rows[i], CIMA.ROWS.PERSONAL_START + i);
    }
  }
  return null;
}

function personalRowToObject_(row, sheetRow) {
  return {
    sheetRow,
    dni: normalizeDni_(row[0]),
    name: norm_(row[1]),
    phone: norm_(row[2]),
    personalEmail: norm_(row[3]),
    corporateEmail: norm_(row[4]),
    schedule: norm_(row[5]),
    joinDate: row[6],
    modality: norm_(row[7]),
    administrativeStatus: norm_(row[8]),
    supervisor: norm_(row[9]),
    turnId: norm_(row[10]),
    start: row[11],
    end: row[12],
    segment: norm_(row[13]),
    completeness: row[14],
    cimaStatus: norm_(row[15]),
  };
}

function getPersonalPublic_(dni) {
  const p = findPersonalByDni_(dni);
  if (!p) return null;
  return {
    dni: p.dni,
    name: p.name,
    schedule: p.schedule,
    modality: p.modality,
    segment: p.segment,
    turnId: p.turnId,
    start: p.start,
    end: p.end,
    status: p.cimaStatus || p.administrativeStatus,
  };
}

function refreshPersonalRow_(rowNumber) {
  const sh = sheet_(CIMA.SHEETS.PERSONAL);
  if (rowNumber < CIMA.ROWS.PERSONAL_START) return;
  const row = sh.getRange(rowNumber, 1, 1, 10).getValues()[0];
  const dni = normalizeDni_(row[0]);
  const name = norm_(row[1]);
  const phone = norm_(row[2]);
  const personalEmail = norm_(row[3]);
  const corpEmail = norm_(row[4]);
  const schedule = norm_(row[5]);
  const joinDate = row[6];
  const modality = norm_(row[7]);
  const adminStatus = norm_(row[8]);
  const supervisor = norm_(row[9]);

  let turnId = '', start = '', end = '', segment = '';
  const turn = getTurnCatalog_().find(t => t.schedule === schedule);
  if (turn) {
    turnId = turn.id;
    start = turn.start;
    end = turn.end;
    segment = turn.id >= 'T09' ? 'PM' : 'AM';
  }

  const requiredOk = !!dni && !!name && /^\d{9}$/.test(phone) && isValidEmail_(corpEmail) && !!schedule && !!joinDate && !!modality && !!adminStatus;
  const emailComplete = !!corpEmail && isValidEmail_(corpEmail);
  const uniqueness = dni ? countDni_(dni) === 1 : false;
  const complete = requiredOk && emailComplete && uniqueness && !!turnId;
  let cimaStatus = '';
  if (dni) {
    if (!complete) cimaStatus = 'INCOMPLETO';
    else if (adminStatus === 'ACTIVO') cimaStatus = 'ACTIVO';
    else cimaStatus = 'INACTIVO';
  }

  sh.getRange(rowNumber, 11, 1, 6).setValues([[turnId, start, end, segment, complete ? '100%' : 'INCOMPLETO', cimaStatus]]);
}

function countDni_(dni) {
  const sh = sheet_(CIMA.SHEETS.PERSONAL);
  const last = sh.getLastRow();
  if (last < CIMA.ROWS.PERSONAL_START) return 0;
  const vals = sh.getRange(CIMA.ROWS.PERSONAL_START, 1, last - CIMA.ROWS.PERSONAL_HEADER, 1).getValues();
  return vals.filter(r => normalizeDni_(r[0]) === dni).length;
}

function onEdit(e) {
  if (!e || !e.range) return;
  const sh = e.range.getSheet();
  if (sh.getName() === CIMA.SHEETS.PERSONAL) {
    if (e.range.getRow() < CIMA.ROWS.PERSONAL_START || e.range.getColumn() > 10) return;
    refreshPersonalRow_(e.range.getRow());
    return;
  }
  if (sh.getName() === CIMA.SHEETS.CIMA && e.range.getColumn() === 9 && e.range.getRow() >= 11 && e.range.getRow() <= 20) {
    const action = norm_(e.range.getValue());
    const requestId = norm_(sh.getRange(e.range.getRow(), 1).getValue());
    const map = { 'APROBAR': CIMA.STATUS.APPROVED, 'RECHAZAR': CIMA.STATUS.REJECTED, 'CANCELAR': CIMA.STATUS.CANCELLED };
    if (action && requestId && map[action]) {
      try {
        changeOperationStatus(requestId, map[action]);
        e.range.clearContent();
        sh.getParent().toast(`${action}: ${requestId}`, 'CIMA', 3);
      } catch (err) {
        e.range.setValue('');
        sh.getParent().toast(err.message, 'CIMA', 5);
      }
    }
  }
}


/*******************************************************
 * 20_Operations.gs
 *******************************************************/

function createOperation(payload) {
  const lock = LockService.getScriptLock();
  lock.waitLock(10000);
  try {
    const data = normalizeOperationPayload_(payload || {});
    validateOperation_(data);
    const person = findPersonalByDni_(data.dni);
    if (!person) throw new Error('DNI no existe en PERSONAL.');
    if (person.administrativeStatus !== 'ACTIVO') throw new Error('El agente no está ACTIVO.');
    if (person.cimaStatus !== 'ACTIVO') throw new Error('La ficha del agente está INCOMPLETA.');

    if (data.type === CIMA.TYPES.OT && duplicateActiveOt_(data.dni, data.date)) {
      throw new Error('Ya existe una OT activa para ese DNI y fecha.');
    }

    const id = buildRequestId_(data.type);
    const row = [
      id, data.date, data.type, person.dni, person.name, person.schedule, person.turnId, person.segment,
      person.modality, data.prePost, data.hours, data.newSchedule, data.reason, data.counterpart,
      data.company, data.currentDayOff, data.newDayOff, data.adminEvent, CIMA.STATUS.PENDING,
      now_(), getCurrentActor_(), now_(), getCurrentActor_(), data.observation, false,
      JSON.stringify({ source: 'HTML', fields: data.raw })
    ];
    const sh = sheet_(CIMA.SHEETS.OPERACIONES);
    const targetRow = Math.max(sh.getLastRow() + 1, CIMA.ROWS.OPS_START);
    sh.getRange(targetRow, 1, 1, row.length).setValues([row]);
    return { ok: true, requestId: id, status: CIMA.STATUS.PENDING };
  } finally {
    lock.releaseLock();
  }
}

function normalizeOperationPayload_(p) {
  return {
    type: norm_(p.type),
    dni: normalizeDni_(p.dni),
    date: parseDate_(p.date),
    prePost: norm_(p.prePost),
    hours: Number(p.hours || 0),
    newSchedule: norm_(p.newSchedule),
    reason: norm_(p.reason),
    counterpart: norm_(p.counterpart),
    company: norm_(p.company),
    currentDayOff: parseOptionalDate_(p.currentDayOff),
    newDayOff: parseOptionalDate_(p.newDayOff),
    adminEvent: norm_(p.adminEvent),
    observation: norm_(p.observation),
    raw: p,
  };
}

function validateOperation_(d) {
  const types = getOperationTypes_().map(x => x.type);
  if (!types.includes(d.type)) throw new Error('Tipo de operación no permitido.');
  if (!isValidDni_(d.dni)) throw new Error('DNI inválido.');
  if (!(d.date instanceof Date) || isNaN(d.date)) throw new Error('Fecha inválida.');
  if (d.type === CIMA.TYPES.OT) {
    if (!['PRE','POST'].includes(d.prePost)) throw new Error('Selecciona PRE o POST.');
    const max = Number(getConfig_()['Máx OT diaria'] || 0);
    if (!Number.isFinite(d.hours) || d.hours <= 0 || d.hours > max) throw new Error(`Horas OT fuera de rango. Máximo: ${max}.`);
    if (Number(getConfig_()['Horas enteras']) === 1 && !Number.isInteger(d.hours)) throw new Error('La OT debe registrarse en horas enteras.');
  }
  if (d.type === CIMA.TYPES.SHIFT && !getTurnCatalog_().some(t => t.schedule === d.newSchedule)) {
    throw new Error('El nuevo horario no pertenece al catálogo.');
  }
  if (d.type === CIMA.TYPES.SWAP) {
    if (!d.counterpart) throw new Error('Indica la contraparte.');
    if (!(d.currentDayOff instanceof Date) || !(d.newDayOff instanceof Date)) throw new Error('Completa los descansos.');
  }
  if (d.type === CIMA.TYPES.ADMIN && !d.adminEvent) throw new Error('Selecciona el evento administrativo.');
}

function duplicateActiveOt_(dni, date) {
  if (Number(getConfig_()['Una OT por persona/día']) !== 1) return false;
  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const last = sh.getLastRow();
  if (last < CIMA.ROWS.OPS_START) return false;
  const vals = sh.getRange(CIMA.ROWS.OPS_START, 1, last - CIMA.ROWS.OPS_HEADER, CIMA.OPS.STATUS).getValues();
  const target = dateKey_(date);
  return vals.some(r => normalizeDni_(r[CIMA.OPS.DNI-1]) === dni && norm_(r[CIMA.OPS.TYPE-1]) === CIMA.TYPES.OT && dateKey_(r[CIMA.OPS.DATE-1]) === target && [CIMA.STATUS.PENDING,CIMA.STATUS.APPROVED].includes(norm_(r[CIMA.OPS.STATUS-1])));
}

function changeOperationStatus(requestId, nextStatus) {
  const allowed = [CIMA.STATUS.APPROVED, CIMA.STATUS.REJECTED, CIMA.STATUS.CANCELLED];
  if (!allowed.includes(nextStatus)) throw new Error('Estado no permitido.');
  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const row = findOperationRow_(requestId);
  if (!row) throw new Error('Solicitud no encontrada.');
  const current = norm_(sh.getRange(row, CIMA.OPS.STATUS).getValue());
  const valid = (current === CIMA.STATUS.PENDING) || (current === CIMA.STATUS.APPROVED && nextStatus === CIMA.STATUS.CANCELLED);
  if (!valid) throw new Error(`No se puede pasar de ${current} a ${nextStatus}.`);
  sh.getRange(row, CIMA.OPS.STATUS).setValue(nextStatus);
  sh.getRange(row, CIMA.OPS.UPDATED_AT, 1, 2).setValues([[now_(), getCurrentActor_()]]);
  sh.getRange(row, CIMA.OPS.GTR_EXPORT).setValue(nextStatus === CIMA.STATUS.APPROVED);
  refreshDashboard();
  return { ok: true, requestId, status: nextStatus };
}

function findOperationRow_(requestId) {
  const id = norm_(requestId);
  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const last = sh.getLastRow();
  if (last < CIMA.ROWS.OPS_START) return 0;
  const ids = sh.getRange(CIMA.ROWS.OPS_START, CIMA.OPS.REQUEST_ID, last - CIMA.ROWS.OPS_HEADER, 1).getValues();
  const idx = ids.findIndex(r => norm_(r[0]) === id);
  return idx < 0 ? 0 : CIMA.ROWS.OPS_START + idx;
}

function listPendingOperations() {
  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const last = sh.getLastRow();
  if (last < CIMA.ROWS.OPS_START) return [];
  const rows = sh.getRange(CIMA.ROWS.OPS_START, 1, last - CIMA.ROWS.OPS_HEADER, CIMA.OPS.DETAIL).getValues();
  return rows.filter(r => norm_(r[CIMA.OPS.STATUS-1]) === CIMA.STATUS.PENDING)
    .map(r => ({ requestId:r[0], date:r[1], type:r[2], dni:r[3], name:r[4], schedule:r[5], prePost:r[9], hours:r[10], newSchedule:r[11], reason:r[12], status:r[18] }));
}

function buildRequestId_(type) {
  const prefix = ({ HORAS_EXTRA:'HE', CAMBIO_HORARIO:'CH', INTERCAMBIO_DESCANSO:'ID', ADMINISTRATIVO:'AD' })[type] || 'OP';
  return `${prefix}-${Utilities.formatDate(now_(), Session.getScriptTimeZone(), 'yyyyMMdd-HHmmss')}-${Utilities.getUuid().slice(0,6).toUpperCase()}`;
}

function parseDate_(value) {
  if (value instanceof Date && !isNaN(value)) return value;
  const s = norm_(value);
  if (!s) return new Date('invalid');
  const parts = s.includes('/') ? s.split('/') : s.split('-');
  if (parts.length !== 3) return new Date('invalid');
  const [a,b,c] = parts.map(Number);
  if (s.includes('/')) return new Date(c, b - 1, a);
  return new Date(a, b - 1, c);
}

function parseOptionalDate_(value) {
  if (value === '' || value == null) return '';
  const d = parseDate_(value);
  return isNaN(d) ? '' : d;
}

function dateKey_(value) {
  if (!(value instanceof Date)) return '';
  return Utilities.formatDate(value, Session.getScriptTimeZone(), 'yyyy-MM-dd');
}


/*******************************************************
 * 30_Admin.gs
 *******************************************************/

function searchAgentForAdmin(dni) {
  const p = findPersonalByDni_(dni);
  return p ? {
    dni:p.dni, name:p.name, schedule:p.schedule, modality:p.modality, segment:p.segment,
    turnId:p.turnId, status:p.cimaStatus, administrativeStatus:p.administrativeStatus,
  } : null;
}

function registerAdminEvent(payload) {
  if (!isSheetAdmin_()) throw new Error('Acceso administrativo no autorizado.');
  const dni = normalizeDni_(payload?.dni);
  const event = norm_(payload?.adminEvent);
  if (!isValidDni_(dni)) throw new Error('DNI inválido.');
  if (!event) throw new Error('Selecciona el evento.');
  const p = findPersonalByDni_(dni);
  if (!p) throw new Error('DNI no existe en PERSONAL.');
  const id = buildRequestId_(CIMA.TYPES.ADMIN);
  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const row = [
    id, parseDate_(payload.date), CIMA.TYPES.ADMIN, p.dni, p.name, p.schedule, p.turnId, p.segment, p.modality,
    '', '', '', '', norm_(payload.counterpart), norm_(payload.company), parseOptionalDate_(payload.currentDayOff), parseOptionalDate_(payload.newDayOff),
    event, CIMA.STATUS.APPROVED, now_(), getCurrentActor_(), now_(), getCurrentActor_(), norm_(payload.observation), false,
    JSON.stringify({ admin: true })
  ];
  const target = Math.max(sh.getLastRow() + 1, CIMA.ROWS.OPS_START);
  sh.getRange(target, 1, 1, row.length).setValues([row]);
  return { ok:true, requestId:id, event };
}

function isSheetAdmin_() {
  const configured = norm_(getConfig_()['Supervisor email']).toLowerCase();
  const actor = getCurrentActor_().toLowerCase();
  return configured !== '' && actor === configured;
}


/*******************************************************
 * 40_Dashboard.gs
 *******************************************************/

function refreshDashboard() {
  const ss = getBook_();
  const personal = ss.getSheetByName(CIMA.SHEETS.PERSONAL);
  const ops = ss.getSheetByName(CIMA.SHEETS.OPERACIONES);

  const pLast = personal.getLastRow();
  const people = pLast >= CIMA.ROWS.PERSONAL_START
    ? personal.getRange(CIMA.ROWS.PERSONAL_START, 1, pLast - CIMA.ROWS.PERSONAL_HEADER, 16).getValues()
    : [];
  const active = people.filter(r => norm_(r[8]) === 'ACTIVO').length;

  const oLast = ops.getLastRow();
  const rows = oLast >= CIMA.ROWS.OPS_START
    ? ops.getRange(CIMA.ROWS.OPS_START, 1, oLast - CIMA.ROWS.OPS_HEADER, 26).getValues()
    : [];
  const today = dateKey_(new Date());
  const pending = rows.filter(r => norm_(r[18]) === CIMA.STATUS.PENDING);
  const pendingOt = pending.filter(r => norm_(r[2]) === CIMA.TYPES.OT).length;
  const pendingShift = pending.filter(r => norm_(r[2]) === CIMA.TYPES.SHIFT).length;
  const pendingSwap = pending.filter(r => norm_(r[2]) === CIMA.TYPES.SWAP).length;
  const approvedToday = rows.filter(r => norm_(r[18]) === CIMA.STATUS.APPROVED && dateKey_(r[1]) === today).length;
  const rejectedToday = rows.filter(r => norm_(r[18]) === CIMA.STATUS.REJECTED && dateKey_(r[1]) === today).length;
  const todayCount = rows.filter(r => dateKey_(r[1]) === today).length;
  const adminToday = rows.filter(r => norm_(r[2]) === CIMA.TYPES.ADMIN && dateKey_(r[1]) === today).length;
  const complete = people.filter(r => norm_(r[15]) === 'ACTIVO').length;

  const sh = ss.getSheetByName(CIMA.SHEETS.CIMA);
  sh.getRange('B5').setValue(active);
  sh.getRange('D5').setValue(complete);
  sh.getRange('F5').setValue(pendingOt);
  sh.getRange('H5').setValue(rows.filter(r => norm_(r[2]) === CIMA.TYPES.OT && norm_(r[18]) === CIMA.STATUS.APPROVED).length);
  sh.getRange('B6').setValue(adminToday);
  sh.getRange('D6').setValue(approvedToday);
  sh.getRange('F6').setValue(rejectedToday);
  sh.getRange('H6').setValue(todayCount);
  sh.getRange('B10:D13').setValues([
    ['HORAS_EXTRA', pendingOt, rows.filter(r => norm_(r[2]) === CIMA.TYPES.OT && norm_(r[18]) === CIMA.STATUS.APPROVED).length],
    ['CAMBIO_HORARIO', pendingShift, rows.filter(r => norm_(r[2]) === CIMA.TYPES.SHIFT && norm_(r[18]) === CIMA.STATUS.APPROVED).length],
    ['INTERCAMBIO_DESCANSO', pendingSwap, rows.filter(r => norm_(r[2]) === CIMA.TYPES.SWAP && norm_(r[18]) === CIMA.STATUS.APPROVED).length],
    ['TOTAL', pending.length, rows.filter(r => norm_(r[18]) === CIMA.STATUS.APPROVED).length],
  ]);
  sh.getRange('B7').setValue(now_());
  sh.getRange('B7').setNumberFormat('dd/mm/yyyy hh:mm');

  const list = pending.slice(0, 10).map(r => [r[0],r[1],r[2],r[3],r[4],operationDetail_(r),r[18],r[19],'','']);
  while (list.length < 10) list.push(['','','','','','','','','','']);
  sh.getRange('A11:J20').setValues(list);
}

function operationDetail_(r) {
  if (norm_(r[2]) === CIMA.TYPES.OT) return `${r[9] || ''} · ${r[10] || ''} h`;
  if (norm_(r[2]) === CIMA.TYPES.SHIFT) return r[11] || '';
  if (norm_(r[2]) === CIMA.TYPES.SWAP) return r[13] || '';
  return r[17] || '';
}


/*******************************************************
 * 50_Triggers.gs
 *******************************************************/

function installTriggers_() {
  const handlers = ['dailyClose'];
  ScriptApp.getProjectTriggers().forEach(t => {
    if (handlers.includes(t.getHandlerFunction())) ScriptApp.deleteTrigger(t);
  });
  ScriptApp.newTrigger('dailyClose').timeBased().everyMinutes(15).create();
}

function dailyClose() {
  const props = PropertiesService.getScriptProperties();
  const now = new Date();
  const today = dateKey_(now);
  const config = getConfig_();
  const closeHour = 0;
  const currentHour = Number(Utilities.formatDate(now, Session.getScriptTimeZone(), 'H'));
  const currentMinute = Number(Utilities.formatDate(now, Session.getScriptTimeZone(), 'm'));
  if (currentHour !== closeHour || currentMinute > 14) return;
  if (props.getProperty(CIMA.PROPS.LAST_CLOSE) === today) return;
  if (Number(config['Auto-reject al cierre']) !== 1) {
    props.setProperty(CIMA.PROPS.LAST_CLOSE, today);
    return;
  }

  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const last = sh.getLastRow();
  if (last >= CIMA.ROWS.OPS_START) {
    const range = sh.getRange(CIMA.ROWS.OPS_START, 1, last - CIMA.ROWS.OPS_HEADER, 26);
    const values = range.getValues();
    let changed = false;
    values.forEach(r => {
      if (norm_(r[18]) === CIMA.STATUS.PENDING && dateKey_(r[1]) < today) {
        r[18] = CIMA.STATUS.AUTO_REJECTED;
        r[21] = now;
        r[22] = 'SYSTEM_CLOSE';
        changed = true;
      }
    });
    if (changed) range.setValues(values);
  }
  props.setProperty(CIMA.PROPS.LAST_CLOSE, today);
  refreshDashboard();
}

function diagnoseCima() {
  const ss = getBook_();
  const required = Object.values(CIMA.SHEETS);
  const missing = required.filter(n => !ss.getSheetByName(n));
  const result = {
    spreadsheetId: ss.getId(),
    sheets: ss.getSheets().map(s => s.getName()),
    missing,
    triggers: ScriptApp.getProjectTriggers().map(t => t.getHandlerFunction()),
    status: missing.length ? 'ERROR' : 'OK',
  };
  Logger.log(JSON.stringify(result, null, 2));
  return result;
}


/*******************************************************
 * 60_GTR.gs
 *******************************************************/

function exportApprovedGtr() {
  const sh = sheet_(CIMA.SHEETS.OPERACIONES);
  const last = sh.getLastRow();
  if (last < CIMA.ROWS.OPS_START) throw new Error('No hay operaciones.');
  const rows = sh.getRange(CIMA.ROWS.OPS_START, 1, last - CIMA.ROWS.OPS_HEADER, 26).getValues()
    .filter(r => norm_(r[18]) === CIMA.STATUS.APPROVED && r[24] === true);
  if (!rows.length) throw new Error('No hay operaciones aprobadas pendientes de GTR.');

  const out = [['FECHA','DNI','NOMBRE','TIPO','HORARIO','TURNO','SEGMENTO','MODALIDAD','PRE_POST','HORAS','NUEVO_HORARIO','MOTIVO','ESTADO','REQUEST_ID']];
  rows.forEach(r => out.push([r[1],r[3],r[4],r[2],r[5],r[6],r[7],r[8],r[9],r[10],r[11],r[12],r[18],r[0]]));
  const csv = out.map(row => row.map(csvCell_).join(',')).join('\n');
  const folder = getOrCreateFolder_('CIMA_GTR');
  const file = folder.createFile(`GTR_CIMA_${Utilities.formatDate(new Date(), Session.getScriptTimeZone(), 'yyyyMMdd_HHmm')}.csv`, csv, MimeType.CSV);

  rows.forEach(r => {
    const row = findOperationRow_(r[0]);
    if (row) sh.getRange(row, CIMA.OPS.GTR_EXPORT).setValue(false);
  });
  return file.getUrl();
}

function csvCell_(v) {
  if (v == null) return '""';
  const s = v instanceof Date ? Utilities.formatDate(v, Session.getScriptTimeZone(), 'yyyy-MM-dd HH:mm:ss') : String(v);
  return `"${s.replace(/"/g, '""')}"`;
}

function getOrCreateFolder_(name) {
  const it = DriveApp.getFoldersByName(name);
  return it.hasNext() ? it.next() : DriveApp.createFolder(name);
}


/*******************************************************
 * 70_WebApp.gs
 *******************************************************/

function doGet() {
  return HtmlService.createTemplateFromFile('Index')
    .evaluate()
    .setTitle('CIMA | Operación')
    .setXFrameOptionsMode(HtmlService.XFrameOptionsMode.ALLOWALL);
}

function getInitialData() {
  return {
    turns: getTurnCatalog_(),
    operationTypes: getOperationTypes_(),
    actor: getCurrentActor_(),
  };
}

function lookupAgent(dni) {
  return getPersonalPublic_(dni);
}


/*******************************************************
 * 80_Utils.gs
 *******************************************************/

function appHealth() {
  return diagnoseCima();
}
